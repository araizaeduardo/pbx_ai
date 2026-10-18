// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package sys

import (
	"strings"
	"testing"
)

func TestSplitMessageShort(t *testing.T) {
	msg := "Hello world"
	chunks := SplitMessage(msg, 4000)
	if len(chunks) != 1 || chunks[0] != msg {
		t.Fatalf("expected 1 chunk identical to msg, got %v", chunks)
	}
}

func TestSplitMessageEmpty(t *testing.T) {
	chunks := SplitMessage("", 4000)
	if len(chunks) != 0 {
		t.Fatalf("expected empty chunks, got %v", chunks)
	}
}

func TestSplitMessageParagraphs(t *testing.T) {
	p1 := strings.Repeat("a", 100)
	p2 := strings.Repeat("b", 100)
	full := p1 + "\n\n" + p2

	chunks := SplitMessage(full, 120)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	if chunks[0] != p1 {
		t.Errorf("expected chunk 0 to be p1, got %q", chunks[0])
	}
	if chunks[1] != p2 {
		t.Errorf("expected chunk 1 to be p2, got %q", chunks[1])
	}
}

func TestSplitMessageHardCut(t *testing.T) {
	unbroken := strings.Repeat("x", 250)
	chunks := SplitMessage(unbroken, 100)
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks for length 250 with maxLen 100, got %d", len(chunks))
	}
	if len(chunks[0]) != 100 || len(chunks[1]) != 100 || len(chunks[2]) != 50 {
		t.Errorf("unexpected chunk lengths: %d, %d, %d", len(chunks[0]), len(chunks[1]), len(chunks[2]))
	}
}
