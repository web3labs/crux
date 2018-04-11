package utils

import "testing"

func TestBuildUrl(t *testing.T) {
	runUrlTest(t, "http://localhost:9001/", "/endpoint", "http://localhost:9001/endpoint")
	runUrlTest(t, "http://localhost:9001", "/endpoint", "http://localhost:9001/endpoint")
	runUrlTest(t, "http://localhost:9001", "endpoint", "http://localhost:9001/endpoint")
	runUrlTest(t, "http://localhost:9001//", "/endpoint", "http://localhost:9001/endpoint")
}

func runUrlTest(t *testing.T, baseUrl, path, expected string) {
	url, err := BuildUrl(baseUrl, path)

	if err != nil {
		t.Error(err)
	}

	if url != expected {
		t.Errorf("Url created: %s, does not match expected: %s", url, expected)
	}
}