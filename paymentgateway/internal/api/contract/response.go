package contract

type Response struct {
	Code    string `json:"code"`
	Message string `json:"message,omitempty"`
	TrackID string `json:"x_track_id,omitempty"`
	Result  any    `json:"result,omitempty"`
}
