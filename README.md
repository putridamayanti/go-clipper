# Go Clipper ✂️

An automated YouTube video clipper that uses AI (Gemini 1.5 Flash) to identify viral segments and cut them automatically for short-form content.

## Features
- **YouTube Downloader**: Downloads high-quality video using `yt-dlp`.
- **AI Analysis**: Uses Gemini 1.5 Flash to "watch" the video and find high-engagement segments.
- **Auto-Clipping**: Extracts segments using `ffmpeg` with precise timestamps.
- **Local Storage**: Saves all clips locally in your desired output folder.

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
Run the clipper by providing a YouTube URL:
```bash
go run cmd/clipper/main.go -url "https://www.youtube.com/watch?v=your_video_id"
```

### Custom Aspect Ratio
To format clips for TikTok/Shorts (9:16 vertical) or Instagram (1:1 square):
```bash
go run cmd/clipper/main.go -url "..." -ratio 9:16
```
*(Supports: 9:16, 1:1, 4:5, 16:9)*

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
