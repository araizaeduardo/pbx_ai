// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package sys

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/eja/tibula/sys"
)

var Options typeConfigPbx

func Configure() error {
	flag.StringVar(&Options.MediaPath, "media-path", "/tmp/", "Media temporary folder")
	flag.StringVar(&Options.MetaUrl, "meta-url", "", "Meta graph API URL")
	flag.StringVar(&Options.MetaUser, "meta-user", "", "Meta user id")
	flag.StringVar(&Options.MetaAuth, "meta-auth", "", "Meta auth")
	flag.StringVar(&Options.MetaToken, "meta-token", "", "Meta token")
	flag.StringVar(&Options.TelegramToken, "telegram-token", "", "Telegram token")
	flag.StringVar(&Options.AiToken, "ai-token", "", "AI token")
	flag.StringVar(&Options.LlmUrl, "llm-url", "", "LLM API url")
	flag.StringVar(&Options.AsteriskAgi, "asterisk-agi", "127.0.0.1:4573", "Asterisk AGI address")
	flag.StringVar(&Options.AsteriskAri, "asterisk-ari", "http://localhost:8088", "Asterisk ARI URL")
	flag.StringVar(&Options.AsteriskToken, "asterisk-token", "", "Asterisk token")
	flag.StringVar(&Options.Cache, "cache", "/tmp/", "Cache path")
	flag.StringVar(&Options.MailSender, "mail-sender", "", "Mail sender")
	flag.BoolVar(&Options.Asterisk, "asterisk", false, "start the asterisk agi service")
	flag.StringVar(&Options.McpUrl, "mcp-url", "", "Model Context Protocol URL")
	flag.StringVar(&Options.McpToken, "mcp-token", "", "Model Context Protocol token")
	flag.BoolVar(&Options.Chat, "chat", false, "Enable web chat")
	flag.BoolVar(&Options.ChatAudio, "chat-audio", false, "Enable web chat with audio support")
	flag.StringVar(&Options.ChatCors, "chat-cors", "*", "Web chat CORS allowed origin")
	flag.StringVar(&Options.AlertNtfy, "alert-ntfy", "", "Ntfy topic URL for critical failure alerts")
	flag.Usage = Help

	if err := sys.Configure(); err != nil {
		return err
	}
	Options.TypeConfig = sys.Options

	if sys.Commands.Start && sys.Options.ConfigFile == "" {
		sys.Options.ConfigFile = Name + ".json"
		if _, err := os.Stat(sys.Options.ConfigFile); err != nil {
			sys.Options.ConfigFile = ""
		}
	}

	if sys.Options.ConfigFile != "" {
		if err := sys.ConfigRead(sys.Options.ConfigFile, &Options); err != nil {
			return err
		}
	}

	setupLogging(Options.LogLevel)

	if _, err := os.Stat(Options.MediaPath); os.IsNotExist(err) {
		if err := os.MkdirAll(Options.MediaPath, 0755); err != nil {
			return fmt.Errorf("cannot create media folder: %w", err)
		}
	}

	return nil
}

func setupLogging(levelInt int) {
	var level slog.Level
	switch levelInt {
	case 3:
		level = slog.LevelDebug
	case 2:
		level = slog.LevelInfo
	case 1:
		level = slog.LevelWarn
	case 0:
		level = slog.LevelError
	default:
		if levelInt > 3 {
			level = slog.LevelDebug
		} else {
			level = slog.LevelInfo
		}
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
}
