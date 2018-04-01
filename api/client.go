package api

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

type ResendRequest struct {
	Type       string  `json:"type"`
	PublicKey  string  `json:"publicKey"`
	Key        string  `json:"key,omitempty"`
}

type PrivateKeyBytes struct {
	Bytes  string  `json:"bytes"`
}

type PrivateKey struct {
	Data  PrivateKeyBytes  `json:"data"`
	Type  string           `json:"unlocked"`
}
