package dtos

type AnalyzeRequest struct {
	YoutubeUrl      string `json:"youtube_url"`
	VideoPath       string `json:"video_path"`
	Ratio           string `json:"ratio" default:"9:16"` // 9:16
	Count           int    `json:"count" default:"3"`    // Number of clips generated
	MinimumDuration int    `json:"minimum_duration" default:"30"`
	MaximumDuration int    `json:"maximum_duration" default:"60"`
	OutputPath      string `json:"output_path" default:"./output"`

	DownloadVideo bool `json:"download_video"`
	ExtractAudio  bool `json:"extract_audio"`
}

type CutVideoRequest struct {
}
