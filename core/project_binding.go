package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type BindingKey struct {
	Platform string
	UserID   string
	BotID    string
}

func (k BindingKey) String() string {
	return k.Platform + ":" + k.UserID + ":" + k.BotID
}

type BindingRecord struct {
	ActiveProject string    `json:"active_project"`
	SwitchedAt    time.Time `json:"switched_at"`
}

type BindingStore struct {
	mu        sync.RWMutex
	storePath string
	records   map[string]BindingRecord
}

func NewBindingStore(storePath string) *BindingStore {
	s := &BindingStore{
		storePath: storePath,
		records:   make(map[string]BindingRecord),
	}
	if storePath != "" {
		s.load()
	}
	return s
}

func (s *BindingStore) Get(key BindingKey) (BindingRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.records[key.String()]
	return rec, ok
}

func (s *BindingStore) Set(key BindingKey, project string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[key.String()] = BindingRecord{
		ActiveProject: project,
		SwitchedAt:    time.Now(),
	}
	s.saveLocked()
}

func (s *BindingStore) saveLocked() {
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

func (s *BindingStore) load() {
	data, err := os.ReadFile(s.storePath)
	if err != nil {
		return
	}
	var records map[string]BindingRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return
	}
	if records != nil {
		s.records = records
	}
}
