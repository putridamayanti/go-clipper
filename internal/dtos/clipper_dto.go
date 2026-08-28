package dtos

type AnalyzeRequest struct {
	YoutubeUrl      string    `json:"youtube_url"`
	VideoPath       string    `json:"video_path"`
	Ratio           string    `json:"ratio" default:"9:16"` // 9:16
	Count           int       `json:"count" default:"3"`    // Number of clips generated
	MinimumDuration int       `json:"minimum_duration" default:"30"`
	MaximumDuration int       `json:"maximum_duration" default:"60"`
	StartTimestamp  string    `json:"start_timestamp"`
	EndTimestamp    string    `json:"end_timestamp"`
	OutputPath      string    `json:"output_path" default:"./output"`
	Segments        []Segment `json:"segments"`

	DownloadVideo   bool `json:"download_video"`
	IncludeSubtitle bool `json:"include_subtitle"`
	ExtractAudio    bool `json:"extract_audio"`
	WithoutAnalyze  bool `json:"without_analyze"`
	WithCaption     bool `json:"with_caption"`
}

type Segment struct {
	Start       string `json:"start"` // Format: SS
	End         string `json:"end"`   // Format: SS
	Description string `json:"description"`
	Hook        string `json:"hook"`
}

type AnalysisResult struct {
	Description string    `json:"description"`
	Segments    []Segment `json:"segments"`
}

type DescriptionResult struct {
	Description string `json:"description"`
	Tags        string `json:"tags"`
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

type DownloadVideoRequest struct {
	YoutubeUrl      string `json:"youtube_url"`
	IncludeSubtitle bool   `json:"include_subtitle"`
	VideoPath       string `json:"video_path"`
}
