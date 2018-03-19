package utils

import "testing"

func TestNextPowerOf2(t *testing.T) {

	values := map[int]int{
		0: 0,
		1: 1,
		2: 2,
		3: 4,
		4: 4,
		5: 8,
		7: 8,
		8: 8,
		9: 16,
		16: 16,
		17: 32,
		1023: 1024,
		1024: 1024,
		1025: 2048,
	}

	for value, expected := range values {
		if NextPowerOf2(value) != expected {
			t.Errorf("Next power of 2 for %d, does not equal %d", value, expected)
		}
	}
}
