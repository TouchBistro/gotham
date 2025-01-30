package http

type ResponseEnvelop struct {
	Request interface{} `json:"request,omitempty"`
	Params  interface{} `json:"params,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Message *string     `json:"message,omitempty"`
	Code    int64       `json:"returnCode"`
}
