# PBX AI — Configuration & CLI Reference

PBX AI can be configured via:
1. **Command-line flags** (highest priority)
2. **JSON configuration file** (`pbx.json` by default, or specified via `--config <path>`)
3. **Database settings** (`aiSettings` table managed via Web / CLI)

---

## Command-Line Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--start` | bool | `false` | Starts the web server, AGI telephony service, and webhook listeners |
| `--db-setup` | bool | `false` | Creates and initializes database tables, settings, and default prompts |
| `--wizard` | bool | `false` | Interactive command-line setup wizard to generate `pbx.json` and initialize DB |
| `--config` | string | `""` | Path to a custom JSON configuration file |
| `--db-type` | string | `sqlite` | Database engine (`sqlite` or `mysql`) |
| `--db-name` | string | `pbx.db` | Database file name (SQLite) or database name (MySQL) |
| `--db-user` | string | `""` | MySQL user |
| `--db-pass` | string | `""` | MySQL password |
| `--db-host` | string | `localhost` | MySQL host |
| `--db-port` | int | `3306` | MySQL port |
| `--web-port` | int | `35248` | Port for Web Chat, Dashboard, `/health`, `/metrics`, `/meta`, `/tg` |
| `--chat` | bool | `false` | Enable web chat frontend at `/chat` and dashboard at `/dashboard` |
| `--chat-audio` | bool | `false` | Enable voice recording & speech playback in web chat |
| `--chat-cors` | string | `*` | Allowed CORS origins for `/chat` endpoint |
| `--media-path` | string | `/tmp/` | Directory for temporary incoming/outgoing audio files |
| `--cache` | string | `/tmp/` | Directory for caching generated TTS audio files |
| `--log-level` | int | `0` | Logging verbosity (0: Error, 1: Warn, 2: Info, 3: Debug) |
| `--alert-ntfy` | string | `""` | Ntfy topic URL for real-time provider failure push notifications |
| `--ai-token` | string | `""` | OpenAI / LLM API key |
| `--llm-url` | string | `""` | OpenAI-compatible chat completions API URL |
| `--mcp-url` | string | `""` | Model Context Protocol (MCP) server endpoint |
| `--mcp-token` | string | `""` | Authentication token for MCP server |
| `--asterisk` | bool | `false` | Start the FastAGI voice server on `--asterisk-agi` |
| `--asterisk-agi` | string | `127.0.0.1:4573` | Bind address for Asterisk FastAGI voice server |
| `--asterisk-ari` | string | `http://localhost:8088` | Asterisk REST Interface (ARI) URL for PJSIP dynamic sync |
| `--asterisk-token` | string | `""` | Asterisk authentication token (validated via constant-time comparison) |
| `--meta-url` | string | `""` | WhatsApp Cloud Graph API base URL |
| `--meta-user` | string | `""` | WhatsApp Phone Number ID |
| `--meta-auth` | string | `""` | WhatsApp System User Access Token |
| `--meta-token` | string | `""` | Webhook verification token for Meta |
| `--telegram-token` | string | `""` | Telegram Bot Token from @BotFather |
| `--mail-sender` | string | `""` | Sender address for outgoing automated emails |

---

## Configuration File Example (`pbx.json`)

See [pbx.example.json](../pbx.example.json) for a complete template:

```json
{
  "db_type": "sqlite",
  "db_name": "pbx.db",
  "web_port": 35248,
  "chat": true,
  "chat_audio": true,
  "chat_cors": "*",
  "media_path": "/tmp/pbx/media/",
  "cache": "/tmp/pbx/cache/",
  "asterisk": true,
  "asterisk_agi": "127.0.0.1:4573",
  "ai_token": "sk-proj-...",
  "llm_url": "https://api.openai.com/v1/chat/completions",
  "telegram_token": "123456789:ABC...",
  "log_level": 2
}
```
