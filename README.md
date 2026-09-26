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

## How it works
1.  **Extract**: Downloads the video and extracts the audio.
2.  **Analyze**: Uploads the audio to Gemini 1.5 Flash to identify viral moments.
3.  **Process**: FFmpeg cuts the video into multiple short-form clips based on the AI's timestamps.
