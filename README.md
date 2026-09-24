# Go Clipper ✂️

An automated YouTube video clipper that uses AI (Gemini 1.5 Flash) to identify viral segments and cut them automatically for short-form content.

## Features
- **Flexible Source**: Process YouTube links or local video files.
- **YouTube Downloader**: Downloads high-quality video using `yt-dlp`.
- **English Translation**: Automatically translates foreign speech (Korean, Chinese, etc.) to English.
- **Standalone SRT**: Generates a matching `.srt` file for every clip automatically.
- **AI Analysis**: Uses Gemini 1.5 Flash to find high-engagement segments.
- **Auto-Clipping**: Extracts segments using `ffmpeg` with precise timestamps.
- **Optional Hard-Subbing**: Use the `-burn` flag to bake captions into the video.

## Prerequisites
You must have the following tools installed on your system:
1.  **Go** (1.25 or later)
2.  **FFmpeg**: `brew install ffmpeg`
3.  **yt-dlp**: `brew install yt-dlp`

## Setup
1.  Clone the repository.
2.  Install Go dependencies:
    ```bash
    go mod tidy
    ```
3.  Create a `.env` file based on `.env.example` and add your [Gemini API Key](https://aistudio.google.com/app/apikey).
    ```bash
    cp .env .env
    ```

## Usage
Run the clipper by providing a YouTube URL or a local file:
```bash
# From YouTube
go run cmd/clipper/main.go -url "https://www.youtube.com/watch?v=..."

# From Local File
go run cmd/clipper/main.go -input "./videos/my_video.mp4"
```

### Custom Aspect Ratio
To format clips for TikTok/Shorts (9:16 vertical) or Instagram (1:1 square):
```bash
go run cmd/clipper/main.go -url "..." -ratio 9:16
```
*(Supports: 9:16, 1:1, 4:5, 16:9)*

### Number of Clips
To specify exactly how many viral clips Gemini should identify:
```bash
go run cmd/clipper/main.go -url "..." -count 5
```
*(Default: 3)*

### Minimum Duration
To ensure clips are not too short, specify the minimum duration in seconds:
```bash
go run cmd/clipper/main.go -url "..." -min 30
```
*(Default: 15 seconds)*

### Subtitles & Burning
By default, the tool saves a separate `.srt` file next to each clip. If you want to burn (hard-sub) the English captions into the video:
```bash
go run cmd/clipper/main.go -url "..." -burn
```

### Troubleshooting "Too Many Requests" (429)
If you see `HTTP Error 429: Too Many Requests`, it means YouTube is throttling your IP. You can bypass this by passing cookies from your browser:
```bash
go run cmd/clipper/main.go -url "..." -cookies chrome
```
*(Supports: chrome, safari, firefox, edge, opera)*

### Captioner Tool
If you have existing clips in `output/clips` and want to generate SRT subtitles for all of them using Gemini:
```bash
go run cmd/captioner/main.go
```
This tool will:
1. Scan all `.mp4` files in `output/clips`.
2. Upload each video to Gemini.
3. Generate and save a matching `.srt` subtitle file for each clip.

The clips will be saved in the `./output` directory by default. You can change this using the `-out` flag:
```bash
go run cmd/clipper/main.go -url "..." -out "./my_clips"
```

## REST API
The project also ships an HTTP API (built with Gin) that exposes the same pipeline.

### Running the server
The API reads these from your `.env` (or environment):
```env
GEMINI_API_KEY=your-key-for-analysis
GEMINI_API_KEY_FOR_CAPTION=your-key-for-captions
GEMINI_MODEL=                 # optional, default: gemini-3-flash-preview
GEMINI_MODEL_FOR_CAPTION=     # optional, default: same as GEMINI_MODEL
```

Start the server (listens on port `8000`):
```bash
go run cmd/api/main.go
```

All endpoints accept and return JSON. Successful responses wrap the payload in `data`; failures return `{"error": "..."}` with HTTP `400`.

> **Note:** The `default` values on the request structs are **not** applied automatically. Send `count`, `minimum_duration`, `maximum_duration`, `ratio`, and `output_path` explicitly, or they will be `0` / empty.

| Method | Endpoint | Description |
| ------ | -------- | ----------- |
| `GET`  | `/ping` | Health check |
| `POST` | `/api/v1/download` | Download a YouTube video |
| `POST` | `/api/v1/analyze` | Ask Gemini for viral segments (no cutting) |
| `POST` | `/api/v1/generate-description` | Generate a video description and tags |
| `POST` | `/api/v1/clipper` | Full pipeline: analyze → (download) → cut → (caption) |
| `POST` | `/api/v1/clipper/captions` | Generate `.srt` files for existing clips |

---

### `GET /ping`
Health check.

```bash
curl http://localhost:8000/ping
```
```json
{ "message": "pong" }
```

---

### `POST /api/v1/download`
Downloads a YouTube video into `./output` using `yt-dlp`.

| Field | Type | Description |
| ----- | ---- | ----------- |
| `youtube_url` | string | YouTube video URL |
| `include_subtitle` | bool | Also download subtitles |

```bash
curl -X POST http://localhost:8000/api/v1/download \
  -H "Content-Type: application/json" \
  -d '{
    "youtube_url": "https://www.youtube.com/watch?v=VIDEO_ID",
    "include_subtitle": false
  }'
```
```json
{ "data": "output/video_title.mp4" }
```

---

### `POST /api/v1/analyze`
Sends the YouTube URL straight to Gemini and returns the suggested segments. No video is downloaded or cut.

| Field | Type | Description |
| ----- | ---- | ----------- |
| `youtube_url` | string | YouTube video URL |
| `count` | int | Number of segments to find |
| `minimum_duration` | int | Minimum clip length (seconds) |
| `maximum_duration` | int | Maximum clip length (seconds) |
| `start_timestamp` | string | Only analyze from this point (optional, default: start of video) |
| `end_timestamp` | string | Only analyze up to this point (optional, default: end of video) |

```bash
curl -X POST http://localhost:8000/api/v1/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "youtube_url": "https://www.youtube.com/watch?v=VIDEO_ID",
    "count": 3,
    "minimum_duration": 30,
    "maximum_duration": 60,
    "start_timestamp": "00:00",
    "end_timestamp": "10:00"
  }'
```
```json
{
  "data": {
    "description": "Short summary of the video",
    "segments": [
      {
        "start": "30",
        "end": "72",
        "description": "Deep insight about AI",
        "hook": "The Truth About AI"
      }
    ]
  }
}
```

---

### `POST /api/v1/generate-description`
Generates a description and tags for the video (e.g. for uploading the clips).

| Field | Type | Description |
| ----- | ---- | ----------- |
| `youtube_url` | string | YouTube video URL |

```bash
curl -X POST http://localhost:8000/api/v1/generate-description \
  -H "Content-Type: application/json" \
  -d '{ "youtube_url": "https://www.youtube.com/watch?v=VIDEO_ID" }'
```
```json
{
  "data": {
    "description": "Generated description...",
    "tags": "tag1, tag2, tag3"
  }
}
```

---

### `POST /api/v1/clipper`
Runs the full pipeline:
1. Analyzes the video with Gemini (or uses the `segments` you provide when `without_analyze` is `true`).
2. Downloads the video if `download_video` is `true`; otherwise uses `video_path`.
3. Cuts each segment into a clip with FFmpeg and saves it to `output_path`.
4. If `with_caption` is `true`, generates `.srt` files for the clips in `./output/clips`.

| Field | Type | Description |
| ----- | ---- | ----------- |
| `youtube_url` | string | YouTube video URL (used for analysis and download) |
| `video_path` | string | Local source video. **Required** if `download_video` is `false` |
| `download_video` | bool | Download the video with `yt-dlp` before cutting |
| `include_subtitle` | bool | Also download subtitles when downloading |
| `ratio` | string | `9:16`, `1:1`, `4:5`, `16:9` (any other value keeps the original size) |
| `count` | int | Number of segments to find |
| `minimum_duration` | int | Minimum clip length (seconds) |
| `maximum_duration` | int | Maximum clip length (seconds) |
| `start_timestamp` | string | Only analyze from this point (optional) |
| `end_timestamp` | string | Only analyze up to this point (optional) |
| `output_path` | string | Where clips are written, e.g. `./output/clips` |
| `without_analyze` | bool | Skip Gemini and cut the given `segments` instead |
| `segments` | array | Segments to cut (`start`, `end` in seconds, `hook`, `description`). Used with `without_analyze` |
| `with_caption` | bool | Generate `.srt` files for the clips afterwards |

**Example: analyze, download, and cut**
```bash
curl -X POST http://localhost:8000/api/v1/clipper \
  -H "Content-Type: application/json" \
  -d '{
    "youtube_url": "https://www.youtube.com/watch?v=VIDEO_ID",
    "download_video": true,
    "ratio": "9:16",
    "count": 3,
    "minimum_duration": 30,
    "maximum_duration": 60,
    "output_path": "./output/clips",
    "with_caption": true
  }'
```

**Example: cut your own segments from a local file (no Gemini analysis)**
```bash
curl -X POST http://localhost:8000/api/v1/clipper \
  -H "Content-Type: application/json" \
  -d '{
    "video_path": "./output/my_video.mp4",
    "ratio": "9:16",
    "output_path": "./output/clips",
    "without_analyze": true,
    "segments": [
      { "start": "30",  "end": "75",  "hook": "The Truth About AI", "description": "Deep insight about AI" },
      { "start": "120", "end": "170", "hook": "Nobody Expected This", "description": "Plot twist moment" }
    ]
  }'
```

Response (the segments that were used; clips are saved as `<hook>.mp4` in `output_path`):
```json
{
  "data": {
    "description": "Short summary of the video",
    "segments": [
      { "start": "30", "end": "75", "description": "Deep insight about AI", "hook": "The Truth About AI" }
    ]
  }
}
```

---

### `POST /api/v1/clipper/captions`
Generates an `.srt` file next to every `.mp4` in the clips folder using Gemini. Clips that already have an `.srt` are skipped.

| Field | Type | Description |
| ----- | ---- | ----------- |
| `clips_path` | string | Folder containing the clips (default: `./output/clips`) |

```bash
curl -X POST http://localhost:8000/api/v1/clipper/captions \
  -H "Content-Type: application/json" \
  -d '{ "clips_path": "./output/clips" }'
```
```json
{
  "data": [
    "output/clips/the_truth_about_ai.srt",
    "output/clips/nobody_expected_this.srt"
  ]
}
```

## How it works
1.  **Extract**: Downloads the video and extracts the audio.
2.  **Analyze**: Uploads the audio to Gemini 1.5 Flash to identify viral moments.
3.  **Process**: FFmpeg cuts the video into multiple short-form clips based on the AI's timestamps.
