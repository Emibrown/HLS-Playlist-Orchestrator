package orchestrator

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
)

func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	repo := NewInMemoryRepository()
	svc := NewService(repo, 6)
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewHandler(svc, log, nil)
}

func newTestRouter(h *Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Route("/streams/{stream_id}", func(r chi.Router) {
		r.Post("/end", h.EndStream)
		r.Route("/renditions/{rendition}", func(r chi.Router) {
			r.Post("/segments", h.RegisterSegment)
			r.Get("/playlist.m3u8", h.GetPlaylist)
		})
	})
	return r
}

func TestHandler_RegisterSegment(t *testing.T) {
	h := newTestHandler(t)
	r := newTestRouter(h)

	body := map[string]interface{}{"sequence": 42, "duration": 2.0, "path": "/segments/42.ts"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/streams/s1/renditions/720p/segments", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rec.Code)
	}
}

func TestHandler_RegisterSegment_bad_request(t *testing.T) {
	h := newTestHandler(t)
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/streams/s1/renditions/720p/segments", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_RegisterSegment_conflict_after_end(t *testing.T) {
	h := newTestHandler(t)
	r := newTestRouter(h)

	// Register one segment then end stream
	b1, _ := json.Marshal(map[string]interface{}{"sequence": 1, "duration": 2.0, "path": "/1.ts"})
	req1 := httptest.NewRequest(http.MethodPost, "/streams/s1/renditions/720p/segments", bytes.NewReader(b1))
	req1.Header.Set("Content-Type", "application/json")
	rec1 := httptest.NewRecorder()
	r.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusCreated {
		t.Fatalf("setup: expected 201, got %d", rec1.Code)
	}

	reqEnd := httptest.NewRequest(http.MethodPost, "/streams/s1/end", nil)
	recEnd := httptest.NewRecorder()
	r.ServeHTTP(recEnd, reqEnd)
	if recEnd.Code != http.StatusOK {
		t.Fatalf("end stream: expected 200, got %d", recEnd.Code)
	}

	// Try to register another segment
	b2, _ := json.Marshal(map[string]interface{}{"sequence": 2, "duration": 2.0, "path": "/2.ts"})
	req2 := httptest.NewRequest(http.MethodPost, "/streams/s1/renditions/720p/segments", bytes.NewReader(b2))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusConflict {
		t.Errorf("expected 409 after stream ended, got %d", rec2.Code)
	}
}

func TestHandler_GetPlaylist(t *testing.T) {
	h := newTestHandler(t)
	r := newTestRouter(h)

	// Register segments
	for i := 38; i <= 40; i++ {
		b, _ := json.Marshal(map[string]interface{}{"sequence": i, "duration": 2.0, "path": "/segments/" + strconv.Itoa(i) + ".ts"})
		req := httptest.NewRequest(http.MethodPost, "/streams/s1/renditions/720p/segments", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("register %d: expected 201, got %d", i, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/streams/s1/renditions/720p/playlist.m3u8", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "application/vnd.apple.mpegurl" {
		t.Errorf("expected playlist content type, got %s", rec.Header().Get("Content-Type"))
	}
	body := rec.Body.String()
	if !bytes.Contains([]byte(body), []byte("#EXTM3U")) || !bytes.Contains([]byte(body), []byte("#EXT-X-MEDIA-SEQUENCE:38")) {
		t.Errorf("unexpected playlist body: %s", body)
	}
}

func TestHandler_GetPlaylist_not_found(t *testing.T) {
	h := newTestHandler(t)
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/streams/missing/renditions/720p/playlist.m3u8", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandler_EndStream(t *testing.T) {
	h := newTestHandler(t)
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/streams/s1/end", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHandler_GetPlaylist_path_params(t *testing.T) {
	h := newTestHandler(t)
	r := newTestRouter(h)

	// Chi routes use stream_id and rendition; segment path uses them
	b, _ := json.Marshal(map[string]interface{}{"sequence": 1, "duration": 2.0, "path": "/1.ts"})
	req := httptest.NewRequest(http.MethodPost, "/streams/my-stream/renditions/480p/segments", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/streams/my-stream/renditions/480p/playlist.m3u8", nil)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec2.Code)
	}
}
