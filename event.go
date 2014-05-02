package pusher

type Event struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}

// eventError represent a pusher:error data
type eventError struct {
	message string `json:"message"`
	code    int    `json:"code"`
}
