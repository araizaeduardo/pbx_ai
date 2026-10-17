package sys

import (
	"strings"
	"testing"
)

func TestMarkdownToText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "plain text",
			input:    "Hello world",
			contains: []string{"Hello world"},
		},
		{
			name:     "markdown header and bold text",
			input:    "# Title\n\nThis is **bold** text.",
			contains: []string{"Title", "This is", "bold", "text."},
		},
		{
			name:     "code block text extraction",
			input:    "Here is code:\n```go\nfmt.Println(\"test\")\n```",
			contains: []string{"Here is code:", "fmt.Println(\"test\")"},
		},
		{
			name:     "empty input",
			input:    "",
			contains: []string{""},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := MarkdownToText([]byte(tc.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, exp := range tc.contains {
				if !strings.Contains(got, exp) {
					t.Errorf("MarkdownToText: expected to contain %q, got %q", exp, got)
				}
			}
		})
	}
}
