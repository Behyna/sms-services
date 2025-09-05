package contract

type ResponseError struct {
	Successful bool   `json:"successful"`
	Code       any    `json:"code"`
	Message    string `json:"message"`
	Error      string `json:"error"`
}

type Response struct {
	Successful any `json:"successful"`
	Code       any `json:"code"`
	Message    any `json:"message,omitempty"`
	TrackID    any `json:"x_track_id"`
	Result     any `json:"result"`
}
