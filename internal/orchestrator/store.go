package orchestrator

// Store is the persistence abstraction for stream state.
// Implementations can be in-memory, file-based, or remote.
// The Repository uses Store for all reads and writes; callers of Repository
// do not need to know which Store is used.
type Store interface {
	GetStream(id StreamID) (*StreamState, bool)
	SetStream(s *StreamState)
	ListStreamIDs() []StreamID
}

// InMemoryStore is an in-memory implementation of Store.
type InMemoryStore struct {
	streams map[StreamID]*StreamState
}

// NewInMemoryStore returns a new empty in-memory store.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		streams: make(map[StreamID]*StreamState),
	}
}

// GetStream implements Store.GetStream.
func (s *InMemoryStore) GetStream(id StreamID) (*StreamState, bool) {
	st, ok := s.streams[id]
	return st, ok
}

// SetStream implements Store.SetStream.
func (s *InMemoryStore) SetStream(st *StreamState) {
	s.streams[st.ID] = st
}

// ListStreamIDs implements Store.ListStreamIDs.
func (s *InMemoryStore) ListStreamIDs() []StreamID {
	ids := make([]StreamID, 0, len(s.streams))
	for id := range s.streams {
		ids = append(ids, id)
	}
	return ids
}
