// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package db

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eja/pbx/sys"
)

func TestConversationPersistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pbx-db-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dbFile := filepath.Join(tmpDir, "test_conv.db")
	sys.Options.DbType = "sqlite"
	sys.Options.DbName = dbFile

	uid := "test-user-persistent-1"

	// 1. Save messages
	if err := ConversationSave(uid, "user", "Hello bot"); err != nil {
		t.Fatalf("failed to save user message: %v", err)
	}
	if err := ConversationSave(uid, "assistant", "Hello human!"); err != nil {
		t.Fatalf("failed to save assistant message: %v", err)
	}

	// 2. Load messages
	msgs, err := ConversationLoad(uid, 10)
	if err != nil {
		t.Fatalf("failed to load messages: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Content != "Hello bot" {
		t.Errorf("unexpected msg[0]: %+v", msgs[0])
	}
	if msgs[1].Role != "assistant" || msgs[1].Content != "Hello human!" {
		t.Errorf("unexpected msg[1]: %+v", msgs[1])
	}

	// 3. Reset conversation
	if err := ConversationReset(uid); err != nil {
		t.Fatalf("failed to reset conversation: %v", err)
	}

	msgsAfter, err := ConversationLoad(uid, 10)
	if err != nil {
		t.Fatalf("failed to load messages after reset: %v", err)
	}
	if len(msgsAfter) != 0 {
		t.Fatalf("expected 0 messages after reset, got %d", len(msgsAfter))
	}
}
