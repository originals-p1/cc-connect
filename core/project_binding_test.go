package core

import (
	"path/filepath"
	"testing"
)

func TestProjectBindingStoreSetAndGet(t *testing.T) {
	store := NewBindingStore("")
	key := BindingKey{Platform: "telegram", UserID: "u1", BotID: "b1"}

	if rec, ok := store.Get(key); ok || rec.ActiveProject != "" {
		t.Fatalf("Get() before set = (%+v, %v), want empty,false", rec, ok)
	}

	store.Set(key, "repo-a")
	rec, ok := store.Get(key)
	if !ok {
		t.Fatal("Get() ok = false, want true")
	}
	if rec.ActiveProject != "repo-a" {
		t.Fatalf("ActiveProject = %q, want repo-a", rec.ActiveProject)
	}
	if rec.SwitchedAt.IsZero() {
		t.Fatal("SwitchedAt is zero, want timestamp")
	}
}

func TestProjectBindingStoreReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bindings.json")
	key := BindingKey{Platform: "telegram", UserID: "u1", BotID: "b1"}

	store := NewBindingStore(path)
	store.Set(key, "repo-a")

	reloaded := NewBindingStore(path)
	rec, ok := reloaded.Get(key)
	if !ok {
		t.Fatal("Get() ok = false after reload, want true")
	}
	if rec.ActiveProject != "repo-a" {
		t.Fatalf("ActiveProject = %q after reload, want repo-a", rec.ActiveProject)
	}
}

func TestProjectBindingStoreOverwrite(t *testing.T) {
	store := NewBindingStore("")
	key := BindingKey{Platform: "telegram", UserID: "u1", BotID: "b1"}

	store.Set(key, "repo-a")
	store.Set(key, "repo-b")

	rec, ok := store.Get(key)
	if !ok {
		t.Fatal("Get() ok = false, want true")
	}
	if rec.ActiveProject != "repo-b" {
		t.Fatalf("ActiveProject = %q, want repo-b", rec.ActiveProject)
	}
}
