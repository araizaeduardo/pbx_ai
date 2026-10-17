package pbx

import (
	"reflect"
	"testing"
)

func TestTagsExtract(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedText string
		expectedTags []string
	}{
		{
			name:         "no tags",
			input:        "Hello world, how can I help you?",
			expectedText: "Hello world, how can I help you?",
			expectedTags: nil,
		},
		{
			name:         "single language tag",
			input:        "Hola, ¿en qué puedo ayudarte?\n\n[es]",
			expectedText: "Hola, ¿en qué puedo ayudarte?",
			expectedTags: []string{"es"},
		},
		{
			name:         "multiple tags",
			input:        "Operación completada [ntfy:admin] [en]",
			expectedText: "Operación completada",
			expectedTags: []string{"en", "ntfy:admin"},
		},
		{
			name:         "tag with trailing spaces",
			input:        "Goodbye [close]  ",
			expectedText: "Goodbye",
			expectedTags: []string{"close"},
		},
		{
			name:         "bracket inside text is preserved",
			input:        "Item [1] in list [en]",
			expectedText: "Item [1] in list",
			expectedTags: []string{"en"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			text, tags := TagsExtract(tc.input)
			if text != tc.expectedText {
				t.Errorf("TagsExtract text: expected %q, got %q", tc.expectedText, text)
			}
			if len(tc.expectedTags) == 0 && len(tags) == 0 {
				return
			}
			if !reflect.DeepEqual(tags, tc.expectedTags) {
				t.Errorf("TagsExtract tags: expected %v, got %v", tc.expectedTags, tags)
			}
		})
	}
}

func TestFilterLanguage(t *testing.T) {
	tests := []struct {
		name             string
		tags             []string
		defaultLang      string
		expectedLanguage string
	}{
		{
			name:             "detects two-letter language code",
			tags:             []string{"en"},
			defaultLang:      "it",
			expectedLanguage: "en",
		},
		{
			name:             "detects uppercase language code",
			tags:             []string{"ES"},
			defaultLang:      "en",
			expectedLanguage: "es",
		},
		{
			name:             "ignores non-language tags",
			tags:             []string{"close", "sip:100", "ntfy:admin"},
			defaultLang:      "it",
			expectedLanguage: "it",
		},
		{
			name:             "picks language from mixed tags",
			tags:             []string{"close", "de"},
			defaultLang:      "en",
			expectedLanguage: "de",
		},
		{
			name:             "empty tags keeps default",
			tags:             []string{},
			defaultLang:      "en",
			expectedLanguage: "en",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FilterLanguage(tc.tags, tc.defaultLang)
			if got != tc.expectedLanguage {
				t.Errorf("FilterLanguage: expected %q, got %q", tc.expectedLanguage, got)
			}
		})
	}
}
