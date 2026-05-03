package dtos

import "go-clipper/internal/analyzer"

type AnalyzeRequest struct {
	YoutubeUrl      string             `json:"youtube_url"`
	VideoPath       string             `json:"video_path"`
	Ratio           string             `json:"ratio" default:"9:16"` // 9:16
	Count           int                `json:"count" default:"3"`    // Number of clips generated
	MinimumDuration int                `json:"minimum_duration" default:"30"`
	MaximumDuration int                `json:"maximum_duration" default:"60"`
	OutputPath      string             `json:"output_path" default:"./output"`
	Segments        []analyzer.Segment `json:"segments"`

	DownloadVideo  bool `json:"download_video"`
	ExtractAudio   bool `json:"extract_audio"`
	WithoutAnalyze bool `json:"without_analyze"`
}

type CutClipPayload struct {
	SourceVideoPath string `json:"source_video_path"`
	OutputPath      string `json:"output_path"`
	Hook            string `json:"hook"`
	StartSeconds    string `json:"start_seconds"`
	EndSeconds      string `json:"end_seconds"`
}

type GenerateCaptionRequest struct {
	ClipsPath  string   `json:"clips_path"`
	VideosPath []string `json:"videos_path"`
}
