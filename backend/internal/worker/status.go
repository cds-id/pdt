package worker

import (
	"sync"
	"time"
)

type SyncState string

const (
	StateIdle    SyncState = "idle"
	StateSyncing SyncState = "syncing"
)

type SyncInfo struct {
	LastSync  *time.Time `json:"last_sync"`
	NextSync  *time.Time `json:"next_sync"`
	Status    SyncState  `json:"status"`
	LastError *string    `json:"last_error"`
}

type SyncStatus struct {
	mu      sync.RWMutex
	commits map[uint]*SyncInfo
	jira    map[uint]*SyncInfo
}

func NewSyncStatus() *SyncStatus {
	return &SyncStatus{
		commits: make(map[uint]*SyncInfo),
		jira:    make(map[uint]*SyncInfo),
	}
}

func (s *SyncStatus) GetCommitStatus(userID uint) SyncInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if info, ok := s.commits[userID]; ok {
		return *info
	}
	return SyncInfo{Status: StateIdle}
}

func (s *SyncStatus) GetJiraStatus(userID uint) SyncInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if info, ok := s.jira[userID]; ok {
		return *info
	}
	return SyncInfo{Status: StateIdle}
}

func (s *SyncStatus) SetCommitSyncing(userID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.commits[userID]; !ok {
		s.commits[userID] = &SyncInfo{}
	}
	s.commits[userID].Status = StateSyncing
}

func (s *SyncStatus) SetCommitDone(userID uint, nextSync time.Time, syncErr error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.commits[userID]; !ok {
		s.commits[userID] = &SyncInfo{}
	}
	now := time.Now()
	s.commits[userID].LastSync = &now
	s.commits[userID].NextSync = &nextSync
	s.commits[userID].Status = StateIdle
	if syncErr != nil {
		errStr := syncErr.Error()
		s.commits[userID].LastError = &errStr
	} else {
		s.commits[userID].LastError = nil
	}
}

func (s *SyncStatus) SetJiraSyncing(userID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.jira[userID]; !ok {
		s.jira[userID] = &SyncInfo{}
	}
	s.jira[userID].Status = StateSyncing
}

func (s *SyncStatus) SetJiraDone(userID uint, nextSync time.Time, syncErr error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.jira[userID]; !ok {
		s.jira[userID] = &SyncInfo{}
	}
	now := time.Now()
	s.jira[userID].LastSync = &now
	s.jira[userID].NextSync = &nextSync
	s.jira[userID].Status = StateIdle
	if syncErr != nil {
		errStr := syncErr.Error()
		s.jira[userID].LastError = &errStr
	} else {
		s.jira[userID].LastError = nil
	}
}
