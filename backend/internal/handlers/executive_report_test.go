package handlers

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/cds-id/pdt/backend/internal/ai/agent"
	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/services/executive"
)

type fakeCorrelator struct {
	ds  *executive.CorrelatedDataset
	err error
}

func (f *fakeCorrelator) Build(_ context.Context, _ uint, _ *uint, _ executive.DateRange, _ int) (*executive.CorrelatedDataset, error) {
	return f.ds, f.err
}

type scriptedLLM struct {
	events []agent.ExecutiveEvent
}

func (s *scriptedLLM) Stream(_ context.Context, _, _ string, out chan<- agent.ExecutiveEvent) {
	for _, e := range s.events {
		out <- e
	}
	close(out)
}

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("sqlite open: %v", err)
	}
	if err := db.AutoMigrate(&models.ExecutiveReport{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func parseSSEEventNames(body string) []string {
	var events []string
	sc := bufio.NewScanner(strings.NewReader(body))
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "event: ") {
			events = append(events, strings.TrimPrefix(line, "event: "))
		}
	}
	return events
}

func TestExecutiveGenerate_EventOrderAndPersistence(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)

	h := &ExecutiveReportHandler{
		DB:         db,
		Correlator: &fakeCorrelator{ds: &executive.CorrelatedDataset{UserID: 42}},
		Agent: &agent.ExecutiveReportAgent{LLM: &scriptedLLM{events: []agent.ExecutiveEvent{
			{Kind: "delta", Delta: "hello"},
			{Kind: "suggestion", Suggestion: &executive.Suggestion{Kind: "gap", Title: "t", Detail: "d"}},
			{Kind: "done"},
		}}},
	}

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", uint(42)); c.Next() })
	r.POST("/generate", h.Generate)

	body := `{"range_start":"2026-04-01T00:00:00Z","range_end":"2026-04-10T00:00:00Z"}`
	req := httptest.NewRequest("POST", "/generate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	events := parseSSEEventNames(rec.Body.String())
	want := []string{"status", "dataset", "status", "delta", "suggestion", "status", "done"}
	if len(events) != len(want) {
		t.Fatalf("event count: got %v want %v", events, want)
	}
	for i, e := range want {
		if events[i] != e {
			t.Fatalf("event %d: got %s want %s (full: %v)", i, events[i], e, events)
		}
	}

	var row models.ExecutiveReport
	db.First(&row)
	if row.Status != "completed" {
		t.Fatalf("expected status=completed, got %q", row.Status)
	}
	if row.Narrative != "hello" {
		t.Fatalf("expected narrative=hello, got %q", row.Narrative)
	}
}

func TestExecutiveGenerate_AgentFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)

	h := &ExecutiveReportHandler{
		DB:         db,
		Correlator: &fakeCorrelator{ds: &executive.CorrelatedDataset{}},
		Agent: &agent.ExecutiveReportAgent{LLM: &scriptedLLM{events: []agent.ExecutiveEvent{
			{Kind: "error", Err: errors.New("boom")},
		}}},
	}
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", uint(1)); c.Next() })
	r.POST("/generate", h.Generate)

	req := httptest.NewRequest("POST", "/generate",
		bytes.NewBufferString(`{"range_start":"2026-04-01T00:00:00Z","range_end":"2026-04-02T00:00:00Z"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "event: error") {
		t.Fatalf("expected error event, got: %s", rec.Body.String())
	}
	var row models.ExecutiveReport
	db.First(&row)
	if row.Status != "failed" {
		t.Fatalf("expected status=failed, got %q", row.Status)
	}
	if !strings.Contains(row.ErrorMessage, "boom") {
		t.Fatalf("expected error_message to contain boom, got %q", row.ErrorMessage)
	}
}

func TestExecutiveGet_NotOwnerReturns404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	db.Create(&models.ExecutiveReport{UserID: 7, Status: "completed"})

	h := &ExecutiveReportHandler{DB: db}
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", uint(99)); c.Next() })
	r.GET("/executive/:id", h.Get)

	req := httptest.NewRequest("GET", "/executive/1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != 404 {
		t.Fatalf("expected 404 for non-owner, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestExecutiveGenerate_RejectsOversizedRange(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)

	h := &ExecutiveReportHandler{
		DB:         db,
		Correlator: &fakeCorrelator{ds: &executive.CorrelatedDataset{}},
	}
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", uint(1)); c.Next() })
	r.POST("/generate", h.Generate)

	start := time.Now().AddDate(0, 0, -200).Format(time.RFC3339)
	end := time.Now().Format(time.RFC3339)
	body := `{"range_start":"` + start + `","range_end":"` + end + `"}`
	req := httptest.NewRequest("POST", "/generate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400 for oversized range, got %d: %s", rec.Code, rec.Body.String())
	}
}
