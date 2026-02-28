# HLS Playlist Orchestrator

A backend service that manages live HLS playlists: it accepts segment metadata from a transcoder, maintains a sliding window of contiguous segments, and serves valid HLS live playlists. Built with Go and [go-chi](https://github.com/go-chi/chi).

## Features

- **Register segments** (out-of-order and duplicate-safe)
- **Serve live playlists** (contiguous sliding window, no gaps)
- **End stream** (add `#EXT-X-ENDLIST`, reject new segments)
- Configurable sliding window size and port
- Structured logging (slog), graceful shutdown, Prometheus-style metrics
- Docker and docker-compose support

---

## Latency

Assume **segment duration = 2 seconds** and **sliding window = 6 segments**.

**Roughly what is the minimum achievable live latency with this configuration?**

- **About 6–8 seconds** With a segment duration of 2 seconds and a sliding window of 6 segments, the player typically stays about 3 segments behind the live edge for stability. This results in an estimated live latency of roughly 6–8 seconds, since three segments at 2 seconds each give a 6-second baseline, plus additional time for network and player buffering in real-world conditions.

**What changes would reduce latency?**

- **Smaller sliding window** (e.g 2–3 segments) — less for the player to buffer, so it stays closer to the live edge.
- **Shorter segment duration** (e.g 1 s) — lower bound on how fast content can appear; set in the transcoder, not this service.
- **Low-latency HLS (LL-HLS)** Implementing Partial Segments (chunks) so the player can start downloading a segment before it is fully finished.

**What trade-offs would those changes introduce?**

- **Higher CPU/Overhead:** Smaller segments mean the orchestrator handles 2–4x more HTTP requests and playlist updates per minute.
- **Reduced Stability:** A smaller buffer (window) makes the player extremely sensitive to network "jitter." If a single segment is delayed by 1s, the player will immediately buffer (freeze).
- **CDN load:** CDNs are less efficient at caching thousands of tiny files compared to fewer, larger segments, potentially increasing infrastructure costs

---

## Prerequisites

- **Go 1.24+** (for local run)
- **Docker** (optional, for containerized run)

---

## Running Without Docker

### 1. Clone and enter the project

```bash
cd hls-orchestrator
```

### 2. Install dependencies

```bash
go mod download
```

### 3. (Optional) Configure environment

Copy the example env file and edit as needed:

```bash
cp .env.example .env
```

### 4. Run the server

```bash
go run ./cmd/server
```

Or with explicit env vars (no `.env` file):

```bash
PORT=3000 LOG_LEVEL=debug go run ./cmd/server
```

### 5. Stop

Press **Ctrl+C**. The server shuts down gracefully (drains in-flight requests).

---

## Running With Docker

### Docker Compose (recommended)

Uses `.env` for configuration.

```bash
# Build and run in foreground
docker compose up --build

# Run in background
docker compose up -d --build

# Stop
docker compose down
```

The app listens on the port from `PORT` in `.env` (default 8080). Use `http://localhost:8080` (or your `PORT`) for requests.

---

## Configuration

| Variable              | Default | Description                          |
|-----------------------|--------|--------------------------------------|
| `PORT`                | 8080   | HTTP server port                     |
| `SLIDING_WINDOW_SIZE` | 6      | Max segments in the live playlist    |
| `LOG_LEVEL`           | info   | debug, info, warn, error             |
| `LOG_FORMAT`         | json   | json or text                         |

Without a `.env` file, defaults apply. With Docker Compose, set `env_file: .env` (already configured).

---

## API Documentation

Base URL: `http://localhost:8080` (or your `PORT`).

---

### 1. Register Segment

Registers a new segment for a stream/rendition. Segments may arrive out of order; duplicate sequence numbers are ignored. After a stream is ended, new segments are rejected.

**Endpoint**

```
POST /streams/{stream_id}/renditions/{rendition}/segments
```

**Path parameters**

| Name     | Type   | Description                    |
|----------|--------|--------------------------------|
| stream_id| string | Identifier of the stream       |
| rendition| string | Rendition name (e.g. 720p, 480p)|

**Request body** (JSON)

| Field    | Type   | Required | Description              |
|----------|--------|----------|--------------------------|
| sequence | number | yes      | Monotonic segment number |
| duration | number | yes      | Duration in seconds      |
| path     | string | yes      | Path to the .ts file     |

**Example**

```bash
curl -X POST http://localhost:8080/streams/my-stream/renditions/720p/segments \
  -H "Content-Type: application/json" \
  -d '{"sequence": 42, "duration": 2.0, "path": "/segments/42.ts"}'
```

**Responses**

| Code | Description                    |
|------|--------------------------------|
| 201  | Segment registered              |
| 400  | Bad request (missing/invalid body or path params) |
| 409  | Stream or rendition already ended |
| 500  | Internal error                  |

---

### 2. Get Playlist

Returns the HLS live playlist for a stream/rendition (contiguous sliding window, no gaps).

**Endpoint**

```
GET /streams/{stream_id}/renditions/{rendition}/playlist.m3u8
```

**Path parameters**

| Name      | Type   | Description                    |
|-----------|--------|--------------------------------|
| stream_id | string | Identifier of the stream       |
| rendition | string | Rendition name (e.g. 720p)     |

**Example**

```bash
curl http://localhost:8080/streams/my-stream/renditions/720p/playlist.m3u8
```

**Response**

- **200** – Content-Type: `application/vnd.apple.mpegurl`, body is the m3u8 playlist (e.g. `#EXTM3U`, `#EXT-X-VERSION:3`, `#EXT-X-TARGETDURATION`, `#EXT-X-MEDIA-SEQUENCE`, segment list; if stream ended, `#EXT-X-ENDLIST`).
- **404** – Stream or rendition not found.
- **400** – Missing path parameters.

**Example playlist body**

```m3u8
#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:2
#EXT-X-MEDIA-SEQUENCE:40

#EXTINF:2.0,
/segments/40.ts
#EXTINF:2.0,
/segments/41.ts
```

---

### 3. End Stream

Marks a stream as ended. After this, playlists for that stream include `#EXT-X-ENDLIST` and new segments are rejected.

**Endpoint**

```
POST /streams/{stream_id}/end
```

**Path parameters**

| Name      | Type   | Description              |
|-----------|--------|--------------------------|
| stream_id | string | Identifier of the stream |

**Example**

```bash
curl -X POST http://localhost:8080/streams/my-stream/end
```

**Responses**

| Code | Description     |
|------|-----------------|
| 200  | Stream ended   |
| 400  | Missing stream_id |
| 500  | Internal error |

---

### 4. Metrics (Prometheus)

Prometheus-style metrics for the orchestrator.

**Endpoint**

```
GET /metrics
```

**Example**

```bash
curl http://localhost:8080/metrics
```

**Response**

- **200** – Plain text, Prometheus exposition format.

**Metrics**

| Metric                         | Type    | Description                    |
|--------------------------------|---------|--------------------------------|
| `hls_requests_total`           | counter | Total HTTP requests            |
| `hls_segments_registered_total`| counter | Segments successfully registered |
| `hls_streams_ended_total`      | counter | Streams ended                  |
| `hls_active_streams`           | gauge   | Streams not ended              |
| `hls_errors_total`             | counter | Responses with status 4xx/5xx  |

---