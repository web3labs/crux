package utils

import "net/http"

// HttpClient is an interface for sending synchronous HTTP requests.
type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}
