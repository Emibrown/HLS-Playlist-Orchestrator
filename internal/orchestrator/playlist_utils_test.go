package orchestrator

import (
	"strings"
	"testing"
)

func TestBuildLivePlaylist_empty_not_ended(t *testing.T) {
	out := BuildLivePlaylist(nil, false)
	if !strings.HasPrefix(out, "#EXTM3U\n") {
		t.Error("expected #EXTM3U header")
	}
	if !strings.Contains(out, "#EXT-X-VERSION:3") {
		t.Error("expected version 3")
	}
	if !strings.Contains(out, "#EXT-X-TARGETDURATION:1") {
		t.Error("expected target duration 1 for empty")
	}
	if !strings.Contains(out, "#EXT-X-MEDIA-SEQUENCE:0") {
		t.Error("expected media sequence 0")
	}
	if strings.Contains(out, "#EXT-X-ENDLIST") {
		t.Error("should not contain ENDLIST when not ended")
	}
}

func TestBuildLivePlaylist_empty_ended(t *testing.T) {
	out := BuildLivePlaylist(nil, true)
	if !strings.Contains(out, "#EXT-X-ENDLIST") {
		t.Error("expected #EXT-X-ENDLIST when ended")
	}
}

func TestBuildLivePlaylist_with_segments(t *testing.T) {
	segs := []Segment{
		{Sequence: 38, Duration: 2.0, Path: "/segments/38.ts"},
		{Sequence: 39, Duration: 2.0, Path: "/segments/39.ts"},
	}
	out := BuildLivePlaylist(segs, false)

	if !strings.Contains(out, "#EXT-X-TARGETDURATION:2") {
		t.Errorf("expected TARGETDURATION 2: %s", out)
	}
	if !strings.Contains(out, "#EXT-X-MEDIA-SEQUENCE:38") {
		t.Errorf("expected MEDIA-SEQUENCE 38: %s", out)
	}
	if !strings.Contains(out, "#EXTINF:2.0,") {
		t.Error("expected EXTINF with duration 2.0")
	}
	if !strings.Contains(out, "/segments/38.ts") || !strings.Contains(out, "/segments/39.ts") {
		t.Errorf("expected segment paths: %s", out)
	}
	if strings.Contains(out, "#EXT-X-ENDLIST") {
		t.Error("should not contain ENDLIST when not ended")
	}
}

func TestBuildLivePlaylist_with_segments_ended(t *testing.T) {
	segs := []Segment{
		{Sequence: 1, Duration: 2.5, Path: "/a.ts"},
	}
	out := BuildLivePlaylist(segs, true)

	if !strings.Contains(out, "#EXT-X-ENDLIST") {
		t.Error("expected #EXT-X-ENDLIST when ended")
	}
	if !strings.Contains(out, "#EXT-X-TARGETDURATION:3") {
		t.Errorf("expected TARGETDURATION 3 (ceil 2.5): %s", out)
	}
	if !strings.Contains(out, "#EXTINF:2.5,") {
		t.Error("expected EXTINF 2.5")
	}
}

func TestBuildLivePlaylist_target_duration_ceiling(t *testing.T) {
	segs := []Segment{
		{Sequence: 1, Duration: 1.1, Path: "/a.ts"},
	}
	out := BuildLivePlaylist(segs, false)
	if !strings.Contains(out, "#EXT-X-TARGETDURATION:2") {
		t.Errorf("expected TARGETDURATION 2 (ceil 1.1): %s", out)
	}
}
