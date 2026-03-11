package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type LastActiveSessionRecord struct {
	Platform   string    `json:"platform"`
	SessionKey string    `json:"session_key"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type LastActiveSessionStore struct {
	mu        sync.RWMutex
	storePath string
	records   map[string]LastActiveSessionRecord
}

func NewLastActiveSessionStore(storePath string) *LastActiveSessionStore {
	s := &LastActiveSessionStore{
		storePath: storePath,
		records:   make(map[string]LastActiveSessionRecord),
	}
	if storePath != "" {
		s.load()
	}
	return s
}

func (s *LastActiveSessionStore) Get(botID string) (LastActiveSessionRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.records[botID]
	return rec, ok
}

func (s *LastActiveSessionStore) Set(botID, platform, sessionKey string) {
	if botID == "" || platform == "" || sessionKey == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[botID] = LastActiveSessionRecord{
		Platform:   platform,
		SessionKey: sessionKey,
		UpdatedAt:  time.Now(),
	}
	s.saveLocked()
}

func (s *LastActiveSessionStore) saveLocked() {
	if s.storePath == "" {
		return
	}
	data, err := json.MarshalIndent(s.records, "", "  ")
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(s.storePath), 0o755); err != nil {
		return
	}
	_ = AtomicWriteFile(s.storePath, data, 0o644)
}

func (s *LastActiveSessionStore) load() {
	data, err := os.ReadFile(s.storePath)
	if err != nil {
		return
	}
	var records map[string]LastActiveSessionRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return
	}
	if records != nil {
		s.records = records
	}
}
