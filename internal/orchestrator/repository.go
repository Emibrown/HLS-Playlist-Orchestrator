package orchestrator

import (
	"errors"
	"sort"
	"sync"
	"time"
)

// Repository defines the concurrency-safe contract for accessing and mutating
// in-memory stream state.
type Repository interface {
	// RegisterSegment records a new segment for the given stream and rendition.
	// If the stream or rendition does not exist they are created.
	// Duplicate sequence numbers are ignored and do not corrupt state.
	// If the stream or rendition has been ended, an error is returned.
	RegisterSegment(streamID StreamID, renditionID RenditionID, seg Segment) error

	// GetRenditionSnapshot returns an ordered snapshot of all segments for the
	// given stream and rendition, sorted by sequence number, along with the
	// rendition's ended flag. The ok return is false if either the stream or
	// rendition does not exist.
	GetRenditionSnapshot(streamID StreamID, renditionID RenditionID) (segments []Segment, ended bool, ok bool)

	// EndStream marks a stream (and all its renditions) as ended. After this,
	// new segments for the stream will be rejected.
	EndStream(streamID StreamID) error

	// ActiveStreamCount returns the number of streams that are not ended.
	// Used for metrics.
	ActiveStreamCount() int
}

var (
	// ErrStreamEnded is returned when attempting to register a segment on a
	// stream that has already been ended.
	ErrStreamEnded = errors.New("stream has ended")

	// ErrRenditionEnded is returned when attempting to register a segment on a
	// rendition that has already been ended.
	ErrRenditionEnded = errors.New("rendition has ended")
)

// InMemoryRepository is a concurrency-safe in-memory implementation of Repository.
// It uses a Store for persistence; by default that is an InMemoryStore.
type InMemoryRepository struct {
	mu    sync.RWMutex
	store Store
}

// NewInMemoryRepository constructs a new repository with a default in-memory store.
func NewInMemoryRepository() *InMemoryRepository {
	return NewInMemoryRepositoryWithStore(NewInMemoryStore())
}

// NewInMemoryRepositoryWithStore constructs a repository that uses the given Store.
// Useful for testing or for plugging in a different persistence backend.
func NewInMemoryRepositoryWithStore(store Store) *InMemoryRepository {
	return &InMemoryRepository{store: store}
}

// RegisterSegment implements Repository.RegisterSegment.
func (r *InMemoryRepository) RegisterSegment(streamID StreamID, renditionID RenditionID, seg Segment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	stream := r.getOrCreateStreamLocked(streamID)
	if stream.Ended {
		return ErrStreamEnded
	}

	rendition := r.getOrCreateRenditionLocked(stream, renditionID)
	if rendition.Ended {
		return ErrRenditionEnded
	}

	// Ignore duplicate sequence numbers to avoid corrupting state.
	if _, exists := rendition.Segments[seg.Sequence]; exists {
		return nil
	}

	seg.ReceivedAt = time.Now().UTC()
	rendition.Segments[seg.Sequence] = seg

	return nil
}

// GetRenditionSnapshot implements Repository.GetRenditionSnapshot.
func (r *InMemoryRepository) GetRenditionSnapshot(streamID StreamID, renditionID RenditionID) (segments []Segment, ended bool, ok bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stream, exists := r.store.GetStream(streamID)
	if !exists {
		return nil, false, false
	}

	rendition, exists := stream.Renditions[renditionID]
	if !exists {
		return nil, false, false
	}

	// Build a sorted copy of the segments to avoid exposing internal maps.
	if len(rendition.Segments) == 0 {
		return nil, rendition.Ended, true
	}

	sequences := make([]int64, 0, len(rendition.Segments))
	for seq := range rendition.Segments {
		sequences = append(sequences, seq)
	}
	sort.Slice(sequences, func(i, j int) bool { return sequences[i] < sequences[j] })

	segments = make([]Segment, 0, len(sequences))
	for _, seq := range sequences {
		segments = append(segments, rendition.Segments[seq])
	}

	return segments, rendition.Ended, true
}

// EndStream implements Repository.EndStream.
func (r *InMemoryRepository) EndStream(streamID StreamID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	stream, exists := r.store.GetStream(streamID)
	if !exists {
		// Treat ending a non-existent stream as a no-op for idempotency.
		return nil
	}

	if stream.Ended {
		return nil
	}

	stream.Ended = true
	for _, rendition := range stream.Renditions {
		rendition.Ended = true
	}

	return nil
}

// ActiveStreamCount implements Repository.ActiveStreamCount.
func (r *InMemoryRepository) ActiveStreamCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	n := 0
	for _, id := range r.store.ListStreamIDs() {
		if st, ok := r.store.GetStream(id); ok && !st.Ended {
			n++
		}
	}
	return n
}

// getOrCreateStreamLocked returns an existing stream or creates a new one.
// Caller must hold r.mu in write mode.
func (r *InMemoryRepository) getOrCreateStreamLocked(streamID StreamID) *StreamState {
	if stream, ok := r.store.GetStream(streamID); ok {
		return stream
	}

	stream := &StreamState{
		ID:         streamID,
		Renditions: make(map[RenditionID]*RenditionState),
	}
	r.store.SetStream(stream)
	return stream
}

// getOrCreateRenditionLocked returns an existing rendition or creates a new one.
// Caller must hold r.mu in write mode.
func (r *InMemoryRepository) getOrCreateRenditionLocked(stream *StreamState, renditionID RenditionID) *RenditionState {
	if rendition, ok := stream.Renditions[renditionID]; ok {
		return rendition
	}

	rendition := &RenditionState{
		ID:       renditionID,
		Segments: make(map[int64]Segment),
	}
	stream.Renditions[renditionID] = rendition
	return rendition
}
