# Changelog

All notable changes to the **PBX AI** project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [5.0.0] - 2026-10-18

### Added
- **Operations & Telephony Dashboard:** Dark-mode web interface (`/dashboard` and `/pbx/dashboard.html`) featuring real-time KPIs, active session drawer inspection, channel distribution graphs, recent audit log viewer, and provider status indicators.
- **Prometheus Metrics:** Standard `/metrics` endpoint exporting `pbx_requests_total`, `pbx_tokens_total`, `pbx_request_duration_ms_total`, `pbx_active_users`, `pbx_uptime_seconds`, and `pbx_consecutive_errors`.
- **Health Check Endpoint:** JSON status endpoint at `/health` verifying database connectivity, uptime, active sessions, and enabled channels for Docker and Kubernetes probes.
- **Extended `aiLogs` Table:** Schema migration adding `channel`, `tokens`, `latency_ms`, `model`, and `error` columns to track end-to-end interactions across voice and messaging.
- **Persistent Chat History:** New `aiConversations` table preserving multi-turn context across server restarts with automatic chronological restoration.
- **Real-Time SSE Streaming:** Server-Sent Events support on `/chat?stream=true` with word-by-word streaming in `script.js` and live Markdown sanitization.
- **LLM Retry Engine:** Exponential backoff retry loop (up to 3 attempts) handling transient network drops, rate limits (`429 Too Many Requests`), and server overloads (`5xx`), respecting `Retry-After` headers.
- **Long Message Chunking:** New `SplitMessage` utility splitting long AI replies into safe segments under 4096 characters on paragraph or sentence boundaries for WhatsApp and Telegram.
- **WhatsApp & Telegram Inbound Media:** Automatic download and contextual processing for incoming photos and documents (PDF, text).
- **ElevenLabs TTS Support:** Dedicated `elevenlabs` package supporting custom voice IDs and multilingual speech synthesis.
- **MCP Auto-Reconnection:** Client cache eviction and automatic reconnection for Model Context Protocol tools and prompts.
- **Provider Failure Alerting:** Push notification integration with `sys.Ntfy` triggered when consecutive external provider errors occur.
- **Deployment Infrastructure:** Multi-stage `Dockerfile` with `ffmpeg` runtime and healthchecks, plus `docker-compose.yml` for containerized environments.
- **Documentation Suite:** Architectural specifications, Mermaid sequence diagrams, configuration references, Asterisk dialplan guides, and webhook setup instructions in `docs/`.

### Security
- **HMAC-SHA256 Signature Validation:** Mandatory cryptographic signature checks on incoming Meta WhatsApp webhooks (`X-Hub-Signature-256`).
- **Telegram Secret Token Verification:** Webhook validation using `X-Telegram-Bot-Api-Secret-Token`.
- **Timing Attack Prevention:** Asterisk tokens and webhook secrets compared exclusively via `subtle.ConstantTimeCompare`.
- **DOMPurify Sanitization:** Embedded DOMPurify v3.2.4 client-side library preventing stored XSS from untrusted LLM outputs.
- **Rate Limiting:** Thread-safe sliding-window rate limiters protecting `/chat`, `/meta`, and `/tg` endpoints against brute-force and DoS.
- **JSON Request Limiting:** Enforced 1MB `http.MaxBytesReader` protecting JSON parsing from heap exhaustion.
- **CORS Hardening:** Configurable `--chat-cors` flag replacing permissive defaults.

### Fixed
- Thread-safe chat history with mutex bounds protecting against concurrent map read/write panics under high load.
- SQL injection risk in `UserUpdate` with column whitelisting and parameterized queries.
- Asterisk AGI DTMF extraction bug handling response fields and negative timeout codes.
- ffprobe metadata snake_case key checks (`codec_name`, `sample_rate`).
- Telegram and OpenAI API HTTP response status code verification preventing silent failures.
- Shared package-level `http.Client` instances preventing socket leaks.

---

## [4.7.10] - Initial Release
- Initial Asterisk FastAGI, WhatsApp, and Telegram communication bridge with Tibula database integration.
