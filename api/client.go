package api

// SendRequest sends a new transaction to the enclave for storage and propagation to the provided
// recipients.
type SendRequest struct {
	// Payload is the transaction payload data we wish to store.
	Payload string `json:"payload"`
	// From is the sender node identification.
	From string `json:"from"`
	// To is a list of the recipient nodes that should be privy to this transaction payload.
	To []string `json:"to"`
}

// SendResponse is the response to the SendRequest
type SendResponse struct {
	// Key is the key that can be used to retrieve the submitted transaction.
	Key string `json:"key"`
}

// ReceiveRequest
type ReceiveRequest struct {
	Key string `json:"key"`
	To  string `json:"to"`
}

// ReceiveResponse returns the raw payload associated with the ReceiveRequest.
type ReceiveResponse struct {
	Payload string `json:"payload"`
}

// DeleteRequest deletes the entry matching the given key from the enclave.
type DeleteRequest struct {
	Key string `json:"key"`
}

// ResendRequest is used to resend previous transactions.
// There are two types of supported request.
// 1. All transactions associated with a node, in which case the Key field should be omitted.
// 2. A specific transaction with the given key value.
type ResendRequest struct {
	// Type is the resend request type. It should be either "all" or "individual" depending on if
	// you want to request an individual transaction, or all transactions associated with a node.
	Type      string `json:"type"`
	PublicKey string `json:"publicKey"`
	Key       string `json:"key,omitempty"`
}

type UpdatePartyInfo struct {
	Url        string            `json:"url"`
	Recipients map[string][]byte `json:"recipients"`
	Parties    map[string]bool   `json:"parties"`
}

type PartyInfoResponse struct {
	Payload []byte `json:"payload"`
}
type PrivateKeyBytes struct {
	Bytes string `json:"bytes"`
}

// PrivateKey is a container for a private key.
type PrivateKey struct {
	Data PrivateKeyBytes `json:"data"`
	Type string          `json:"type"`
}
