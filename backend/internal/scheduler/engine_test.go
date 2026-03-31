package scheduler

import "testing"

func TestEngineCreation(t *testing.T) {
	e := NewEngine(nil, nil, nil)
	if e == nil {
		t.Fatal("expected non-nil engine")
	}
	if e.pool == nil {
		t.Error("expected pool to be initialized")
	}
	if e.agents == nil {
		t.Error("expected agents map to be initialized")
	}
}
