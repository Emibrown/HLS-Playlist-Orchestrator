package orchestrator

import (
	"errors"
	"strings"
	"testing"
)

func TestNewService_defaultWindowSize(t *testing.T) {
	repo := NewInMemoryRepository()
	svc := NewService(repo, 0)
	if svc == nil {
		t.Fatal("NewService(0) should not be nil")
	}
	// Default window is 6; register 7 contiguous segments, playlist should have last 6
	for i := int64(1); i <= 7; i++ {
		_ = repo.RegisterSegment("s1", "720p", Segment{Sequence: i, Duration: 2.0, Path: "/a.ts"})
	}
	m3u8, ok := svc.GetPlaylist("s1", "720p")
	if !ok {
		t.Fatal("GetPlaylist: ok false")
	}
	if !strings.Contains(m3u8, "#EXT-X-MEDIA-SEQUENCE:2") {
		t.Errorf("default window 6 should show sequence 2..7, got media sequence in: %s", m3u8)
	}
}

func TestService_RegisterSegment(t *testing.T) {
	repo := NewInMemoryRepository()
	svc := NewService(repo, 6)

	err := svc.RegisterSegment("s1", "720p", Segment{Sequence: 1, Duration: 2.0, Path: "/1.ts"})
	if err != nil {
		t.Fatalf("RegisterSegment: %v", err)
	}
	segments, _, ok := repo.GetRenditionSnapshot("s1", "720p")
	if !ok || len(segments) != 1 {
		t.Errorf("expected 1 segment, got ok=%v len=%d", ok, len(segments))
	}
}

func TestService_RegisterSegment_after_end(t *testing.T) {
	repo := NewInMemoryRepository()
	svc := NewService(repo, 6)
	_ = svc.RegisterSegment("s1", "720p", Segment{Sequence: 1, Duration: 2.0, Path: "/1.ts"})
	_ = svc.EndStream("s1")

	err := svc.RegisterSegment("s1", "720p", Segment{Sequence: 2, Duration: 2.0, Path: "/2.ts"})
	if !errors.Is(err, ErrStreamEnded) {
		t.Errorf("expected ErrStreamEnded, got %v", err)
	}
}

func TestService_GetPlaylist_not_found(t *testing.T) {
	repo := NewInMemoryRepository()
	svc := NewService(repo, 6)

	_, ok := svc.GetPlaylist("missing", "720p")
	if ok {
		t.Error("expected ok false for missing stream")
	}
}

func TestService_GetPlaylist_contiguous_window(t *testing.T) {
	repo := NewInMemoryRepository()
	svc := NewService(repo, 6)

	for i := int64(1); i <= 3; i++ {
		_ = svc.RegisterSegment("s1", "720p", Segment{Sequence: i, Duration: 2.0, Path: "/" + string(rune('0'+i)) + ".ts"})
	}

	m3u8, ok := svc.GetPlaylist("s1", "720p")
	if !ok {
		t.Fatal("GetPlaylist: ok false")
	}
	if !strings.Contains(m3u8, "#EXT-X-MEDIA-SEQUENCE:1") {
		t.Errorf("expected media sequence 1: %s", m3u8)
	}
	if !strings.Contains(m3u8, "/1.ts") || !strings.Contains(m3u8, "/3.ts") {
		t.Errorf("expected segments 1..3 in playlist: %s", m3u8)
	}
}

func TestService_GetPlaylist_hide_segments_after_gap(t *testing.T) {
	repo := NewInMemoryRepository()
	svc := NewService(repo, 6)

	_ = svc.RegisterSegment("s1", "720p", Segment{Sequence: 1, Duration: 2.0, Path: "/1.ts"})
	_ = svc.RegisterSegment("s1", "720p", Segment{Sequence: 2, Duration: 2.0, Path: "/2.ts"})
	_ = svc.RegisterSegment("s1", "720p", Segment{Sequence: 4, Duration: 2.0, Path: "/4.ts"})
	_ = svc.RegisterSegment("s1", "720p", Segment{Sequence: 5, Duration: 2.0, Path: "/5.ts"})

	m3u8, ok := svc.GetPlaylist("s1", "720p")
	if !ok {
		t.Fatal("GetPlaylist: ok false")
	}
	// Contiguous from start: only 1,2 (hide 4,5 until 3 arrives) so player never sees a gap
	if !strings.Contains(m3u8, "#EXT-X-MEDIA-SEQUENCE:1") {
		t.Errorf("expected media sequence 1 (contiguous prefix): %s", m3u8)
	}
	if !strings.Contains(m3u8, "/1.ts") || !strings.Contains(m3u8, "/2.ts") {
		t.Errorf("expected /1.ts and /2.ts: %s", m3u8)
	}
	if strings.Contains(m3u8, "/4.ts") || strings.Contains(m3u8, "/5.ts") {
		t.Errorf("should hide segments after gap (4,5) until 3 arrives: %s", m3u8)
	}
}

func TestService_GetPlaylist_window_size_cap(t *testing.T) {
	repo := NewInMemoryRepository()
	svc := NewService(repo, 3)

	for i := int64(1); i <= 6; i++ {
		_ = svc.RegisterSegment("s1", "720p", Segment{Sequence: i, Duration: 2.0, Path: "/x.ts"})
	}

	m3u8, ok := svc.GetPlaylist("s1", "720p")
	if !ok {
		t.Fatal("GetPlaylist: ok false")
	}
	if !strings.Contains(m3u8, "#EXT-X-MEDIA-SEQUENCE:4") {
		t.Errorf("window 3 should show sequences 4,5,6: %s", m3u8)
	}
	// Should have exactly 3 #EXTINF lines
	count := strings.Count(m3u8, "#EXTINF")
	if count != 3 {
		t.Errorf("expected 3 segments in window, got %d", count)
	}
}

func TestService_GetPlaylist_ended_includes_endlist(t *testing.T) {
	repo := NewInMemoryRepository()
	svc := NewService(repo, 6)
	_ = svc.RegisterSegment("s1", "720p", Segment{Sequence: 1, Duration: 2.0, Path: "/1.ts"})
	_ = svc.EndStream("s1")

	m3u8, ok := svc.GetPlaylist("s1", "720p")
	if !ok {
		t.Fatal("GetPlaylist: ok false")
	}
	if !strings.Contains(m3u8, "#EXT-X-ENDLIST") {
		t.Errorf("ended stream should include #EXT-X-ENDLIST: %s", m3u8)
	}
}

func TestService_EndStream(t *testing.T) {
	repo := NewInMemoryRepository()
	svc := NewService(repo, 6)
	_ = svc.RegisterSegment("s1", "720p", Segment{Sequence: 1, Duration: 2.0, Path: "/1.ts"})

	err := svc.EndStream("s1")
	if err != nil {
		t.Fatalf("EndStream: %v", err)
	}
	_, ended, _ := repo.GetRenditionSnapshot("s1", "720p")
	if !ended {
		t.Error("rendition should be ended")
	}
}
