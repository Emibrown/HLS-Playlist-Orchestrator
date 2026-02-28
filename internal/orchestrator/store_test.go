package orchestrator

import (
	"testing"
)

func TestInMemoryStore_GetSetStream(t *testing.T) {
	store := NewInMemoryStore()

	_, ok := store.GetStream(StreamID("s1"))
	if ok {
		t.Error("expected not found for empty store")
	}

	st := &StreamState{
		ID:         StreamID("s1"),
		Renditions: make(map[RenditionID]*RenditionState),
	}
	store.SetStream(st)

	got, ok := store.GetStream(StreamID("s1"))
	if !ok || got != st {
		t.Errorf("GetStream: ok=%v, got %p want %p", ok, got, st)
	}
}

func TestInMemoryStore_SetStream_replaces(t *testing.T) {
	store := NewInMemoryStore()
	st1 := &StreamState{ID: StreamID("s1"), Renditions: make(map[RenditionID]*RenditionState)}
	st2 := &StreamState{ID: StreamID("s1"), Renditions: make(map[RenditionID]*RenditionState)}
	store.SetStream(st1)
	store.SetStream(st2)

	got, ok := store.GetStream(StreamID("s1"))
	if !ok || got != st2 {
		t.Errorf("SetStream should replace: got %p want %p", got, st2)
	}
}

func TestNewInMemoryRepositoryWithStore(t *testing.T) {
	// Verify repository works with an explicitly injected store (persistence abstraction).
	store := NewInMemoryStore()
	repo := NewInMemoryRepositoryWithStore(store)

	err := repo.RegisterSegment(StreamID("s1"), RenditionID("720p"), Segment{Sequence: 1, Duration: 2.0, Path: "/1.ts"})
	if err != nil {
		t.Fatalf("RegisterSegment: %v", err)
	}

	segments, ended, ok := repo.GetRenditionSnapshot(StreamID("s1"), RenditionID("720p"))
	if !ok || ended || len(segments) != 1 {
		t.Errorf("GetRenditionSnapshot: ok=%v ended=%v len=%d", ok, ended, len(segments))
	}

	// State should be in the store we injected
	st, ok := store.GetStream(StreamID("s1"))
	if !ok || st == nil {
		t.Error("injected store should contain stream after RegisterSegment")
	}
}
