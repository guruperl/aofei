package uploaded

import (
	"testing"
)

// TestUploads tests the conversion of WUploads to strings and back.
func TestUploads(t *testing.T) {
	tests := []struct {
		input    WUploads
		expected []string
	}{
		{1, []string{"All"}},
		{2, []string{"buyeruid"}},
		{4, []string{"userid"}},
		{6, []string{"buyeruid", "userid"}},
		{8, []string{"ip"}},
		{10, []string{"buyeruid", "ip"}},
		{12, []string{"userid", "ip"}},
		{14, []string{"buyeruid", "userid", "ip"}},
		{16, []string{"ifa"}},
	}

	for _, test := range tests {
		result := test.input.ToStrings()
		if len(result) != len(test.expected) {
			t.Errorf("input %d : Expected %v but got %v", test.input, test.expected, result)
			continue
		}
		for i := range result {
			if result[i] != test.expected[i] {
				t.Errorf("Expected %v but got %v", test.expected, result)
				break
			}
		}
	}
}
