package asterisk

import (
	"testing"
)

func TestDtmfExtract(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "digit 1 (ASCII 49)",
			input:    "200 result=49 (1)",
			expected: "1",
		},
		{
			name:     "digit 0 (ASCII 48)",
			input:    "200 result=48 (0)",
			expected: "0",
		},
		{
			name:     "digit 9 (ASCII 57)",
			input:    "200 result=57 (9)",
			expected: "9",
		},
		{
			name:     "star key (ASCII 42)",
			input:    "200 result=42 (*)",
			expected: "*",
		},
		{
			name:     "pound key (ASCII 35)",
			input:    "200 result=35 (#)",
			expected: "#",
		},
		{
			name:     "no digit pressed (result 0)",
			input:    "200 result=0",
			expected: "",
		},
		{
			name:     "negative result timeout (result -1)",
			input:    "200 result=-1",
			expected: "",
		},
		{
			name:     "short error response",
			input:    "510 error",
			expected: "",
		},
		{
			name:     "empty response",
			input:    "",
			expected: "",
		},
		{
			name:     "almost 12 characters",
			input:    "200 result=",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := dtmfExtract(tc.input)
			if got != tc.expected {
				t.Errorf("dtmfExtract(%q): expected %q, got %q", tc.input, tc.expected, got)
			}
		})
	}
}
