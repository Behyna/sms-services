package contract

type ResponseError struct {
	Successful bool   `json:"successful"`
	Code       any    `json:"code"`
	Message    string `json:"message"`
	Error      string `json:"error"`
}

type Response struct {
	Successful bool   `json:"successful"`
	Code       string `json:"code"`
	Message    string `json:"message,omitempty"`
	TrackID    string `json:"x_track_id"`
	Result     any    `json:"result"`
}
