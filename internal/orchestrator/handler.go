package orchestrator

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"hls-orchestrator/internal/platform/metrics"

	"github.com/go-chi/chi/v5"
)

const playlistContentType = "application/vnd.apple.mpegurl"

// Handler exposes orchestrator HTTP endpoints using go-chi.
type Handler struct {
	svc     *Service
	log     *slog.Logger
	metrics *metrics.Metrics
}

// NewHandler returns a Handler that uses the given Service, Logger, and optional Metrics.
// Metrics may be nil to disable metric recording (e.g. in tests).
func NewHandler(svc *Service, log *slog.Logger, m *metrics.Metrics) *Handler {
	return &Handler{svc: svc, log: log, metrics: m}
}

// RegisterSegment handles POST /streams/{stream_id}/renditions/{rendition}/segments.
// Body: { "sequence": 42, "duration": 2.0, "path": "/segments/42.ts" }.
func (h *Handler) RegisterSegment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	streamID := StreamID(chi.URLParam(r, "stream_id"))
	renditionID := RenditionID(chi.URLParam(r, "rendition"))

	if streamID == "" || renditionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var seg Segment
	if err := json.NewDecoder(r.Body).Decode(&seg); err != nil {
		h.log.Debug("invalid segment body", slog.String("error", err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := h.svc.RegisterSegment(streamID, renditionID, seg); err != nil {
		switch err {
		case ErrStreamEnded, ErrRenditionEnded:
			h.log.Info("segment rejected stream or rendition ended",
				slog.String("stream_id", string(streamID)),
				slog.String("rendition", string(renditionID)),
				slog.Int64("sequence", seg.Sequence),
				slog.String("error", err.Error()))
			w.WriteHeader(http.StatusConflict)
			return
		default:
			h.log.Error("register segment failed", slog.String("error", err.Error()))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	h.log.Debug("segment registered",
		slog.String("stream_id", string(streamID)),
		slog.String("rendition", string(renditionID)),
		slog.Int64("sequence", seg.Sequence))
	w.WriteHeader(http.StatusCreated)
	if h.metrics != nil {
		h.metrics.IncSegmentsRegistered()
	}
}

// GetPlaylist handles GET /streams/{stream_id}/renditions/{rendition}/playlist.m3u8.
func (h *Handler) GetPlaylist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	streamID := StreamID(chi.URLParam(r, "stream_id"))
	renditionID := RenditionID(chi.URLParam(r, "rendition"))

	if streamID == "" || renditionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	m3u8, ok := h.svc.GetPlaylist(streamID, renditionID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", playlistContentType)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(m3u8))
}

// EndStream handles POST /streams/{stream_id}/end.
func (h *Handler) EndStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	streamID := StreamID(chi.URLParam(r, "stream_id"))
	if streamID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := h.svc.EndStream(streamID); err != nil {
		h.log.Error("end stream failed", slog.String("stream_id", string(streamID)), slog.String("error", err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.log.Info("stream ended", slog.String("stream_id", string(streamID)))
	w.WriteHeader(http.StatusOK)
	if h.metrics != nil {
		h.metrics.IncStreamsEnded()
	}
}
