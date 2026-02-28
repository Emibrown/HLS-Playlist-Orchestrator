package orchestrator

import (
	"sort"
)

// DefaultWindowSize is the default number of segments in the sliding window (per spec).
const DefaultWindowSize = 6

// Service applies business logic (contiguous sliding window) and delegates storage to Repository.
type Service struct {
	repo       Repository
	windowSize int
}

// NewService returns a Service that uses repo and keeps at most windowSize segments
// in the contiguous sliding window. If windowSize <= 0, DefaultWindowSize is used.
func NewService(repo Repository, windowSize int) *Service {
	if windowSize <= 0 {
		windowSize = DefaultWindowSize
	}
	return &Service{repo: repo, windowSize: windowSize}
}

// RegisterSegment records a segment for the given stream and rendition.
// It delegates to the repository; duplicates are idempotent.
func (s *Service) RegisterSegment(streamID StreamID, renditionID RenditionID, seg Segment) error {
	return s.repo.RegisterSegment(streamID, renditionID, seg)
}

// GetPlaylist returns the HLS playlist for the given stream and rendition:
// a contiguous sliding window of at most s.windowSize segments, no gaps.
func (s *Service) GetPlaylist(streamID StreamID, renditionID RenditionID) (m3u8 string, ok bool) {
	segments, ended, ok := s.repo.GetRenditionSnapshot(streamID, renditionID)
	if !ok {
		return "", false
	}
	// This is the alternative implementation to contiguousSlidingWindow.
	window := contiguousVisibleSegments(segments, s.windowSize)
	return BuildLivePlaylist(window, ended), true
}

// EndStream marks the stream as ended; new segments will be rejected.
func (s *Service) EndStream(streamID StreamID) error {
	return s.repo.EndStream(streamID)
}

// contiguousSlidingWindow returns at most windowSize segments
// Avoids players entering an error state when they see e.g. 42 followed by 44.
// segs must be sorted by Sequence ascending.
func contiguousSlidingWindow(segs []Segment, windowSize int) []Segment {
	if windowSize <= 0 || len(segs) == 0 {
		return nil
	}
	// Build contiguous run from the start; stop at first gap.
	out := make([]Segment, 0, windowSize)
	for i := 0; i < len(segs); i++ {
		if i > 0 && segs[i].Sequence != segs[i-1].Sequence+1 {
			break
		}
		out = append(out, segs[i])
	}
	// Sliding window: keep at most the last windowSize segments of that run.
	if len(out) > windowSize {
		out = out[len(out)-windowSize:]
	}
	return out
}

// GetVisibleSegments implements the "Slide then Filter" logic.
// alternative implementation to contiguousSlidingWindow.
func contiguousVisibleSegments(segs []Segment, windowSize int) []Segment {
	if len(segs) == 0 {
		return nil
	}

	// Sort a copy to avoid side effects on the original slice if needed.
	sort.Slice(segs, func(i, j int) bool {
		return segs[i].Sequence < segs[j].Sequence
	})

	// This ensures that even if a segment is missing, it eventually "falls off" the back.
	start := 0
	if len(segs) > windowSize {
		start = len(segs) - windowSize
	}
	windowed := segs[start:]

	// We only include subsequent segments if they are exactly Sequence + 1.
	visible := make([]Segment, 0, len(windowed))
	for i := 0; i < len(windowed); i++ {
		if i > 0 {
			if windowed[i].Sequence != windowed[i-1].Sequence+1 {
				// GAP DETECTED! Stop here.
				break
			}
		}
		visible = append(visible, windowed[i])
	}

	return visible
}
