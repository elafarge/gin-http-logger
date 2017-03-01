package ginfluentd

// Format of a Request body and it's metadata
type HttpContent struct {
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
	Content  string `json:"content"`
}

// Request log format
type RequestLogEntry struct {
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	HTTPVersion string            `json:"http_version"`
	Headers     map[string]string `json:"headers"`
	HeaderSize  int               `json:"headers_size"`
	Content     HttpContent       `json:"content"`
}

// Response log format
type ResponseLogEntry struct {
	Status     int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	HeaderSize int               `json:"headers_size"`
	Content    HttpContent       `json:"content"`
}

// Log format for the whole request
type FluentdLogLine struct {
	Env           string           `json:"fluentd_env"`
	TimeStarted   string           `json:"@timestamp"`
	ClientAddress string           `json:"x_client_address"`
	Time          int64            `json:"duration"`
	Request       RequestLogEntry  `json:"request"`
	Response      ResponseLogEntry `json:"response"`
	Errors        string           `json:"errors"`
}
