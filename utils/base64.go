package utils

import "encoding/base64"

func DecodeBase64(value string) ([]byte, error) {
	dest := make([]byte, base64.StdEncoding.DecodedLen(len(value)))
	n, err := base64.StdEncoding.Decode(dest, []byte(value))
	return dest[:n], err
}