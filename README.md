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
4.  **gallery-dl** (only for Instagram images): `pip install gallery-dl`

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

## AI Provider
Set `AI_PROVIDER` in `.env` to `gemini` (default) or `openai`. See `.env.example` for all variables.

| Variable | Provider | Notes |
|---|---|---|
| `GEMINI_API_KEY`, `GEMINI_API_KEY_FOR_CAPTION` | gemini | Required |
| `GEMINI_MODEL` | gemini | Default `gemini-3-flash-preview` |
| `OPENAI_API_KEY` | openai | Required |
| `OPENAI_MODEL` | openai | Required. Chat model for segments and descriptions |
| `OPENAI_TRANSCRIBE_MODEL` | openai | Default `whisper-1`. Must support SRT output |
| `OPENAI_BASE_URL` | openai | Default `https://api.openai.com/v1` |

Gemini watches the YouTube video directly. OpenAI can't watch video, so it reads a timestamped transcript instead: the video's YouTube subtitles when available, otherwise an OpenAI transcription of the audio (max 25 MB, roughly 1h40m; use `start_timestamp`/`end_timestamp` for longer videos). AI captions on OpenAI use the translation endpoint, which returns English SRT.

## Usage
Start the API server (it listens on port 8000):
```bash
go run ./cmd/api
```
All endpoints take a JSON body and live under `/api/v1`. The examples use `curl`. In PowerShell, call `curl.exe`, or use `Invoke-RestMethod`:
```powershell
$body = @{ youtube_url = "https://www.youtube.com/watch?v=VIDEO_ID"; include_subtitle = $true } | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri http://localhost:8000/api/v1/download -ContentType "application/json" -Body $body
```

> Unset fields are zero, not defaults. Always send `count`, `minimum_duration`, `maximum_duration`, `ratio` and `output_path` when making clips.

### Make clips from a YouTube video
`POST /api/v1/clipper` downloads the video, asks the AI for the best segments, and cuts one clip per segment:
```bash
curl -X POST http://localhost:8000/api/v1/clipper \
  -H "Content-Type: application/json" \
  -d '{
    "youtube_url": "https://www.youtube.com/watch?v=VIDEO_ID",
    "download_video": true,
    "include_subtitle": true,
    "with_caption": true,
    "count": 3,
    "minimum_duration": 30,
    "maximum_duration": 60,
    "ratio": "9:16",
    "output_path": "./output/clips"
  }'
```
Response:
```json
{
  "data": {
    "description": "Short summary of the video",
    "segments": [
      { "start": "30", "end": "75", "description": "Deep insight about AI", "hook": "The Truth About AI" }
    ]
  },
  "clips": [
    { "clip_path": "output/clips/the_truth_about_ai.mp4", "srt_path": "output/clips/the_truth_about_ai.srt", "subtitle_source": "youtube" }
  ]
}
```

| Field | Notes |
|---|---|
| `youtube_url` | Video to analyze. Gemini watches it directly |
| `download_video` | `true` downloads the video first. If `false`, set `video_path` to a local copy |
| `video_path` | Local video to cut, used when `download_video` is `false` |
| `include_subtitle` | Download the video's English YouTube subtitles and slice one `.srt` per clip |
| `with_caption` | Generate AI subtitles for each clip, only when YouTube has none. `subtitle_source` says which was used |
| `count` | Number of clips to find |
| `minimum_duration`, `maximum_duration` | Clip length in seconds |
| `start_timestamp`, `end_timestamp` | Only look for clips inside this part of the video (in seconds) |
| `ratio` | `9:16`, `1:1`, `4:5` or `16:9`. Leave empty to keep the original size |
| `output_path` | Folder for the clips. Each clip is named after its hook |

### Cut clips from segments you choose
Skip the AI and cut exact timestamps (in seconds) from a video you already have:
```bash
curl -X POST http://localhost:8000/api/v1/clipper \
  -H "Content-Type: application/json" \
  -d '{
    "video_path": "./output/My Video.mp4",
    "without_analyze": true,
    "ratio": "9:16",
    "output_path": "./output/clips",
    "segments": [
      { "start": "120", "end": "165", "hook": "Best Moment" },
      { "start": "300", "end": "340", "hook": "Funny Part" }
    ]
  }'
```
Every segment needs `start`, `end` and `hook`; incomplete segments are skipped.

### Download a YouTube video
```bash
curl -X POST http://localhost:8000/api/v1/download \
  -H "Content-Type: application/json" \
  -d '{"youtube_url": "https://www.youtube.com/watch?v=VIDEO_ID", "include_subtitle": true}'
```
Response (`subtitle_path` is empty when the video has no English subtitles):
```json
{ "data": "output/My Video.mp4", "subtitle_path": "output/My Video.en.srt" }
```

### Find segments without cutting
Same body as `/clipper`; returns the AI's segments only:
```bash
curl -X POST http://localhost:8000/api/v1/analyze \
  -H "Content-Type: application/json" \
  -d '{"youtube_url": "https://www.youtube.com/watch?v=VIDEO_ID", "count": 3, "minimum_duration": 30, "maximum_duration": 60}'
```
Response:
```json
{ "data": { "description": "Short summary of the video", "segments": [ { "start": "30", "end": "75", "description": "...", "hook": "..." } ] } }
```

### Write a post description
```bash
curl -X POST http://localhost:8000/api/v1/generate-description \
  -H "Content-Type: application/json" \
  -d '{"youtube_url": "https://www.youtube.com/watch?v=VIDEO_ID"}'
```
Response:
```json
{ "data": { "description": "...", "tags": "..." } }
```

### Caption existing clips
Generates an AI `.srt` next to every `.mp4` in `clips_path` (default `./output/clips`). Clips that already have an `.srt` are skipped:
```bash
curl -X POST http://localhost:8000/api/v1/clipper/captions \
  -H "Content-Type: application/json" \
  -d '{"clips_path": "./output/clips"}'
```
Response lists the new subtitle files:
```json
{ "data": ["output/clips/the_truth_about_ai.srt"] }
```

### Download Instagram images & carousels
Saves every image of an Instagram post, including all carousel slides, to `./output/instagram` as `{username}_{shortcode}_{slide}.{ext}`:
```bash
curl -X POST http://localhost:8000/api/v1/download/instagram \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.instagram.com/p/SHORTCODE/", "cookies_browser": "firefox", "include_videos": false}'
```
Response, in slide order:
```json
{
  "data": [
    "output/instagram/someuser_SHORTCODE_1.jpg",
    "output/instagram/someuser_SHORTCODE_2.jpg",
    "output/instagram/someuser_SHORTCODE_3.jpg"
  ]
}
```

| Field | Notes |
|---|---|
| `url` | Required. Any post link, including `?img_index=N`; the whole carousel is downloaded |
| `cookies_browser` | Browser where you're logged in to Instagram, which hides most posts from logged-out visitors. On Windows, use `firefox`: current Chrome and Edge encrypt their cookies so other tools can't read them |
| `include_videos` | `true` also saves the video slides of a carousel |

### Errors
Most failed requests return HTTP 400 with the reason:
```json
{ "error": "url must be an Instagram post link" }
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
