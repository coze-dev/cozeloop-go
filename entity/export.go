package entity

type UploadSpan struct {
	StartedATMicros  int64              `json:"started_at_micros"`
	SpanID           string             `json:"span_id"`
	ParentID         string             `json:"parent_id"`
	TraceID          string             `json:"trace_id"`
	DurationMicros   int64              `json:"duration_micros"`
	WorkspaceID      string             `json:"workspace_id"`
	SpanName         string             `json:"span_name"`
	SpanType         string             `json:"span_type"`
	StatusCode       int32              `json:"status_code"`
	Input            string             `json:"input"`
	Output           string             `json:"output"`
	ObjectStorage    string             `json:"object_storage"`
	SystemTagsString map[string]string  `json:"system_tags_string"`
	SystemTagsLong   map[string]int64   `json:"system_tags_long"`
	SystemTagsDouble map[string]float64 `json:"system_tags_double"`
	TagsString       map[string]string  `json:"tags_string"`
	TagsLong         map[string]int64   `json:"tags_long"`
	TagsDouble       map[string]float64 `json:"tags_double"`
	TagsBool         map[string]bool    `json:"tags_bool"`
}

type UploadFile struct {
	TosKey     string
	Data       string
	UploadType UploadType
	TagKey     string
	Name       string
	FileType   string
	SpaceID    string
}

type UploadType int64

const (
	UploadTypeLong          UploadType = 1
	UploadTypeMultiModality UploadType = 2
)
