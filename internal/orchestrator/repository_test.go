package orchestrator

import (
	"errors"
	"testing"
)

func TestInMemoryRepository_RegisterSegment(t *testing.T) {
	repo := NewInMemoryRepository()
	streamID := StreamID("s1")
	renditionID := RenditionID("720p")
	seg := Segment{Sequence: 1, Duration: 2.0, Path: "/segments/1.ts"}

	t.Run("success_creates_stream_and_rendition", func(t *testing.T) {
		err := repo.RegisterSegment(streamID, renditionID, seg)
		if err != nil {
			t.Fatalf("RegisterSegment: %v", err)
		}
		got, ended, ok := repo.GetRenditionSnapshot(streamID, renditionID)
		if !ok {
			t.Fatal("GetRenditionSnapshot: ok false")
		}
		if ended {
			t.Error("ended should be false")
		}
		if len(got) != 1 || got[0].Sequence != 1 || got[0].Path != seg.Path {
			t.Errorf("GetRenditionSnapshot: got %v", got)
		}
	})

	t.Run("duplicate_sequence_idempotent", func(t *testing.T) {
		err := repo.RegisterSegment(streamID, renditionID, seg)
		if err != nil {
			t.Fatalf("duplicate RegisterSegment: %v", err)
		}
		got, _, _ := repo.GetRenditionSnapshot(streamID, renditionID)
		if len(got) != 1 {
			t.Errorf("duplicate should not add segment, got len %d", len(got))
		}
	})

	t.Run("out_of_order_segments", func(t *testing.T) {
		_ = repo.RegisterSegment(streamID, renditionID, Segment{Sequence: 3, Duration: 2.0, Path: "/segments/3.ts"})
		_ = repo.RegisterSegment(streamID, renditionID, Segment{Sequence: 2, Duration: 2.0, Path: "/segments/2.ts"})
		got, _, ok := repo.GetRenditionSnapshot(streamID, renditionID)
		if !ok || len(got) != 3 {
			t.Fatalf("expected 3 segments, got %d, ok=%v", len(got), ok)
		}
		if got[0].Sequence != 1 || got[1].Sequence != 2 || got[2].Sequence != 3 {
			t.Errorf("expected sorted by sequence, got %v", got)
		}
	})
}

func TestInMemoryRepository_RegisterSegment_after_end(t *testing.T) {
	repo := NewInMemoryRepository()
	streamID := StreamID("s2")
	renditionID := RenditionID("720p")

	_ = repo.RegisterSegment(streamID, renditionID, Segment{Sequence: 1, Duration: 2.0, Path: "/a.ts"})
	if err := repo.EndStream(streamID); err != nil {
		t.Fatal(err)
	}

	err := repo.RegisterSegment(streamID, renditionID, Segment{Sequence: 2, Duration: 2.0, Path: "/b.ts"})
	if !errors.Is(err, ErrStreamEnded) {
		t.Errorf("expected ErrStreamEnded, got %v", err)
	}

	got, _, _ := repo.GetRenditionSnapshot(streamID, renditionID)
	if len(got) != 1 {
		t.Errorf("no new segment after end, got %d", len(got))
	}
}

func TestInMemoryRepository_GetRenditionSnapshot(t *testing.T) {
	repo := NewInMemoryRepository()

	t.Run("stream_not_found", func(t *testing.T) {
		_, _, ok := repo.GetRenditionSnapshot(StreamID("missing"), RenditionID("720p"))
		if ok {
			t.Error("expected ok false for missing stream")
		}
	})

	t.Run("rendition_not_found", func(t *testing.T) {
		_ = repo.RegisterSegment(StreamID("s3"), RenditionID("720p"), Segment{Sequence: 1, Duration: 2.0, Path: "/x.ts"})
		_, _, ok := repo.GetRenditionSnapshot(StreamID("s3"), RenditionID("480p"))
		if ok {
			t.Error("expected ok false for missing rendition")
		}
	})
}

func TestInMemoryRepository_EndStream(t *testing.T) {
	repo := NewInMemoryRepository()
	streamID := StreamID("s5")

	t.Run("idempotent_nonexistent", func(t *testing.T) {
		if err := repo.EndStream(streamID); err != nil {
			t.Errorf("EndStream nonexistent should be no-op: %v", err)
		}
	})

	_ = repo.RegisterSegment(streamID, RenditionID("720p"), Segment{Sequence: 1, Duration: 2.0, Path: "/a.ts"})
	if err := repo.EndStream(streamID); err != nil {
		t.Fatal(err)
	}

	_, ended, ok := repo.GetRenditionSnapshot(streamID, RenditionID("720p"))
	if !ok || !ended {
		t.Errorf("after EndStream: ok=%v ended=%v", ok, ended)
	}

	t.Run("idempotent_second_call", func(t *testing.T) {
		if err := repo.EndStream(streamID); err != nil {
			t.Errorf("second EndStream should be no-op: %v", err)
		}
	})
}
