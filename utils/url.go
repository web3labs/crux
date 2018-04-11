package utils

import "net/url"

func BuildUrl(rawUrl, rawPath string) (string, error) {
	baseUrl, err := url.Parse(rawUrl)
	if err != nil {
		return "", err
	}

	path, err := url.Parse(rawPath)

	if err != nil {
		return "", err
	}

	return baseUrl.ResolveReference(path).String(), nil
}
