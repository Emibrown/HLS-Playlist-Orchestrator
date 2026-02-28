package orchestrator

import (
	"fmt"
	"math"
	"strings"
)

// BuildLivePlaylist converts a slice of segments (ordered by sequence ascending)
// into a valid HLS live playlist string. If ended is true, #EXT-X-ENDLIST is appended.
// An empty segments slice produces a minimal valid playlist with media sequence 0.
func BuildLivePlaylist(segments []Segment, ended bool) string {
	var b strings.Builder

	b.WriteString("#EXTM3U\n")
	b.WriteString("#EXT-X-VERSION:3\n")

	if len(segments) == 0 {
		b.WriteString("#EXT-X-TARGETDURATION:1\n")
		b.WriteString("#EXT-X-MEDIA-SEQUENCE:0\n")
		if ended {
			b.WriteString("#EXT-X-ENDLIST\n")
		}
		return b.String()
	}

	targetDuration := targetDurationFromSegments(segments)
	mediaSequence := segments[0].Sequence

	b.WriteString(fmt.Sprintf("#EXT-X-TARGETDURATION:%d\n", targetDuration))
	b.WriteString(fmt.Sprintf("#EXT-X-MEDIA-SEQUENCE:%d\n\n", mediaSequence))

	for _, seg := range segments {
		b.WriteString(fmt.Sprintf("#EXTINF:%.1f,\n", seg.Duration))
		b.WriteString(seg.Path)
		b.WriteString("\n")
	}

	if ended {
		b.WriteString("#EXT-X-ENDLIST\n")
	}

	return b.String()
}

// targetDurationFromSegments returns the HLS #EXT-X-TARGETDURATION value:
// the ceiling of the maximum segment duration in seconds (integer).
func targetDurationFromSegments(segments []Segment) int {
	max := 0.0
	for _, seg := range segments {
		if seg.Duration > max {
			max = seg.Duration
		}
	}
	if max <= 0 {
		return 1
	}
	return int(math.Ceil(max))
}
