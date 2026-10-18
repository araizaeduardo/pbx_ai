// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package sys

import (
	"strings"
)

// SplitMessage splits a long string into chunks no larger than maxLen.
// It tries to split on paragraph breaks (\n\n), line breaks (\n), or sentence boundaries (. ),
// and falls back to word boundaries (space) or hard truncation when necessary.
func SplitMessage(text string, maxLen int) []string {
	if maxLen <= 0 {
		maxLen = 4000
	}
	text = strings.TrimSpace(text)
	if len(text) <= maxLen {
		if text == "" {
			return nil
		}
		return []string{text}
	}

	var chunks []string
	remaining := text

	for len(remaining) > 0 {
		if len(remaining) <= maxLen {
			chunks = append(chunks, remaining)
			break
		}

		sub := remaining[:maxLen]
		splitIdx := -1

		// Prefer paragraph break
		if pIdx := strings.LastIndex(sub, "\n\n"); pIdx > maxLen/2 {
			splitIdx = pIdx + 2
		} else if nIdx := strings.LastIndex(sub, "\n"); nIdx > maxLen/2 {
			splitIdx = nIdx + 1
		} else if dotIdx := strings.LastIndex(sub, ". "); dotIdx > maxLen/2 {
			splitIdx = dotIdx + 2
		} else if spaceIdx := strings.LastIndex(sub, " "); spaceIdx > maxLen/3 {
			splitIdx = spaceIdx + 1
		} else {
			// Hard cut
			splitIdx = maxLen
		}

		chunk := strings.TrimSpace(remaining[:splitIdx])
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
		remaining = strings.TrimSpace(remaining[splitIdx:])
	}

	return chunks
}
