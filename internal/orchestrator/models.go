package orchestrator

import "time"

// StreamID uniquely identifies a live stream.
type StreamID string

// RenditionID identifies a particular rendition of a stream (e.g. "720p", "480p").
type RenditionID string

// Segment represents a single HLS media segment.
// This also matches the input JSON payload for registering segments.
type Segment struct {
	Sequence int64   `json:"sequence"`
	Duration float64 `json:"duration"`
	Path     string  `json:"path"`

	// Metadata managed by the orchestrator (not exposed in the API).
	ReceivedAt time.Time `json:"-"` // when this segment was registered
}

// RenditionState holds all in-memory state for a specific rendition of a stream.
type RenditionState struct {
	ID       RenditionID
	Segments map[int64]Segment
	Ended    bool
}

// StreamState is the top-level in-memory representation of a live stream.
type StreamState struct {
	ID         StreamID
	Renditions map[RenditionID]*RenditionState
	Ended      bool
}
