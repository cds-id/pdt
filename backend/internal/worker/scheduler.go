package worker

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"github.com/cds-id/pdt/backend/internal/crypto"
	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/services/storage"
	"gorm.io/gorm"
)

type Scheduler struct {
	DB                 *gorm.DB
	Encryptor          *crypto.Encryptor
	CommitInterval     time.Duration
	JiraInterval       time.Duration
	Status             *SyncStatus
	ReportAutoGenerate bool
	ReportAutoTime     string
	R2                 *storage.R2Client
	commitRunning      atomic.Bool
	jiraRunning        atomic.Bool
	reportRunning      atomic.Bool
	lastReportDate     string
}

func NewScheduler(db *gorm.DB, enc *crypto.Encryptor, commitInterval, jiraInterval time.Duration, reportAutoGen bool, reportAutoTime string, r2 *storage.R2Client) *Scheduler {
	return &Scheduler{
		DB:                 db,
		Encryptor:          enc,
		CommitInterval:     commitInterval,
		JiraInterval:       jiraInterval,
		Status:             NewSyncStatus(),
		ReportAutoGenerate: reportAutoGen,
		ReportAutoTime:     reportAutoTime,
		R2:                 r2,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	log.Printf("[worker] starting scheduler: commits=%s, jira=%s, reports=%v at %s",
		s.CommitInterval, s.JiraInterval, s.ReportAutoGenerate, s.ReportAutoTime)

	go s.commitSyncLoop(ctx)
	go s.jiraSyncLoop(ctx)
	if s.ReportAutoGenerate {
		go s.reportLoop(ctx)
	}
}

func (s *Scheduler) commitSyncLoop(ctx context.Context) {
	s.runCommitSync()

	ticker := time.NewTicker(s.CommitInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[worker] commit sync loop stopped")
			return
		case <-ticker.C:
			s.runCommitSync()
		}
	}
}

func (s *Scheduler) runCommitSync() {
	if !s.commitRunning.CompareAndSwap(false, true) {
		log.Println("[worker] commit sync skipped: previous run still in progress")
		return
	}
	defer s.commitRunning.Store(false)

	log.Println("[worker] commit sync starting")

	var userIDs []uint
	s.DB.Model(&models.Repository{}).Distinct("user_id").Pluck("user_id", &userIDs)

	nextSync := time.Now().Add(s.CommitInterval)

	for _, uid := range userIDs {
		s.Status.SetCommitSyncing(uid)
		results, err := SyncUserCommits(s.DB, s.Encryptor, uid)
		if err != nil {
			s.Status.SetCommitDone(uid, nextSync, err)
			log.Printf("[worker] commit sync failed for user %d: %v", uid, err)
			continue
		}
		for _, r := range results {
			if r.Error != "" {
				log.Printf("[worker] commit sync user=%d repo=%s error=%s", uid, r.RepoName, r.Error)
			} else if r.New > 0 {
				log.Printf("[worker] commit sync user=%d repo=%s new=%d total=%d", uid, r.RepoName, r.New, r.Total)
			}
		}
		s.Status.SetCommitDone(uid, nextSync, nil)
	}

	log.Println("[worker] commit sync completed")
}

func (s *Scheduler) jiraSyncLoop(ctx context.Context) {
	s.runJiraSync()

	ticker := time.NewTicker(s.JiraInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[worker] jira sync loop stopped")
			return
		case <-ticker.C:
			s.runJiraSync()
		}
	}
}

func (s *Scheduler) runJiraSync() {
	if !s.jiraRunning.CompareAndSwap(false, true) {
		log.Println("[worker] jira sync skipped: previous run still in progress")
		return
	}
	defer s.jiraRunning.Store(false)

	log.Println("[worker] jira sync starting")

	var users []models.User
	s.DB.Where("jira_token != '' AND jira_workspace != '' AND jira_email != ''").Find(&users)

	nextSync := time.Now().Add(s.JiraInterval)

	for _, user := range users {
		s.Status.SetJiraSyncing(user.ID)
		err := SyncUserJira(s.DB, s.Encryptor, user.ID)
		if err != nil {
			s.Status.SetJiraDone(user.ID, nextSync, err)
			log.Printf("[worker] jira sync failed for user %d: %v", user.ID, err)
		} else {
			s.Status.SetJiraDone(user.ID, nextSync, nil)
			log.Printf("[worker] jira sync completed for user %d", user.ID)
		}
	}

	log.Println("[worker] jira sync completed")
}

func (s *Scheduler) reportLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[worker] report loop stopped")
			return
		case <-ticker.C:
			s.checkAndGenerateReport()
		}
	}
}

func (s *Scheduler) checkAndGenerateReport() {
	now := time.Now()
	today := now.Format("2006-01-02")

	if s.lastReportDate == today {
		return
	}

	currentTime := now.Format("15:04")
	if currentTime < s.ReportAutoTime {
		return
	}

	if !s.reportRunning.CompareAndSwap(false, true) {
		return
	}
	defer s.reportRunning.Store(false)

	log.Println("[worker] auto-generating daily reports")
	AutoGenerateReports(s.DB, s.Encryptor, s.R2)
	s.lastReportDate = today
	log.Println("[worker] daily report generation completed")
}
