package core

import (
	"fmt"
	"sync"
	"time"
)

type ProjectRuntime struct {
	BotID       string
	Project     ProjectInfo
	Engine      *Engine
	RuntimeData any
	LastUsedAt  time.Time
	CloseFunc   func() error
}

func (r *ProjectRuntime) touch() {
	r.LastUsedAt = time.Now()
}

func (r *ProjectRuntime) close() error {
	if r.CloseFunc != nil {
		return r.CloseFunc()
	}
	return nil
}

type RuntimeFactory func(botID string, proj ProjectInfo) (*ProjectRuntime, error)

type BotRuntimeManager struct {
	mu       sync.Mutex
	catalog  *ProjectCatalog
	maxCache int
	create   RuntimeFactory
	runtimes map[string]map[string]*ProjectRuntime
}

func NewBotRuntimeManager(catalog *ProjectCatalog, maxCache int, create RuntimeFactory) *BotRuntimeManager {
	if maxCache <= 0 {
		maxCache = 1
	}
	return &BotRuntimeManager{
		catalog:  catalog,
		maxCache: maxCache,
		create:   create,
		runtimes: make(map[string]map[string]*ProjectRuntime),
	}
}

func (m *BotRuntimeManager) SetCatalog(catalog *ProjectCatalog) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.catalog = catalog
}

func (m *BotRuntimeManager) GetOrCreate(botID, projectName string) (*ProjectRuntime, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	proj, ok := m.catalog.Projects[projectName]
	if !ok {
		return nil, fmt.Errorf("project %q not found", projectName)
	}

	if m.runtimes[botID] == nil {
		m.runtimes[botID] = make(map[string]*ProjectRuntime)
	}
	if rt, ok := m.runtimes[botID][projectName]; ok {
		rt.touch()
		return rt, nil
	}

	rt, err := m.create(botID, proj)
	if err != nil {
		return nil, err
	}
	if rt == nil {
		return nil, fmt.Errorf("runtime factory returned nil runtime")
	}
	rt.BotID = botID
	rt.Project = proj
	rt.touch()
	m.runtimes[botID][projectName] = rt

	if len(m.runtimes[botID]) > m.maxCache {
		if err := m.evictLRULocked(botID, projectName); err != nil {
			return nil, err
		}
	}

	return rt, nil
}

func (m *BotRuntimeManager) evictLRULocked(botID, skipProject string) error {
	var victimName string
	var victim *ProjectRuntime
	for name, rt := range m.runtimes[botID] {
		if name == skipProject {
			continue
		}
		if victim == nil || rt.LastUsedAt.Before(victim.LastUsedAt) {
			victimName = name
			victim = rt
		}
	}
	if victim == nil {
		return nil
	}
	if err := victim.close(); err != nil {
		return err
	}
	delete(m.runtimes[botID], victimName)
	return nil
}

func (m *BotRuntimeManager) StopAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var firstErr error
	for botID, runtimes := range m.runtimes {
		for name, rt := range runtimes {
			if err := rt.close(); err != nil && firstErr == nil {
				firstErr = fmt.Errorf("stop runtime %s/%s: %w", botID, name, err)
			}
		}
		delete(m.runtimes, botID)
	}
	return firstErr
}
