package api

// TODO: Support protobuf API

type SendRequest struct {
	Payload  string  `json:"payload"`
	From     string  `json:"from"`
	To       []string `json:"to"`
}

type SendResponse struct {
	Key  string  `json:"key"`
}

type ReceiveRequest struct {
	Key  string  `json:"key"`
	To   string  `json:"to"`
}

type ReceiveResponse struct {
	Payload  string  `json:"payload"`
}

type DeleteRequest struct {
	Key  string  `json:"key"`
}
