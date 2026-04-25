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

The clips will be saved in the `./output` directory by default. You can change this using the `-out` flag:
```bash
go run cmd/clipper/main.go -url "..." -out "./my_clips"
```

## How it works
1.  **Extract**: Downloads the video and extracts the audio.
2.  **Analyze**: Uploads the audio to Gemini 1.5 Flash to identify viral moments.
3.  **Process**: FFmpeg cuts the video into multiple short-form clips based on the AI's timestamps.
