# Webhooks Setup Guide: WhatsApp (Meta) & Telegram

PBX AI includes hardened, production-ready webhook listeners for WhatsApp Cloud API and Telegram with cryptographic signature validation and rate limiting.

---

## 1. WhatsApp Cloud API (Meta)

### Webhook Endpoint
- **URL:** `https://your-domain.com/meta`
- **Supported Methods:** `GET` (verification handshake), `POST` (incoming messages)

### Setup in Meta Developer Portal
1. Navigate to your app in the [Meta for Developers Console](https://developers.facebook.com/).
2. Under **WhatsApp > Configuration**:
   - **Callback URL:** `https://your-domain.com/meta`
   - **Verify Token:** Set to the value of `--meta-token` (or `metaToken` in `aiSettings`).
3. Click **Verify and Save**. Meta will send a `GET` request with `hub.mode=subscribe` and your verify token.
4. Subscribe to the `messages` webhook field.

### Cryptographic Security & Signatures
PBX AI verifies the authenticity of incoming messages using the `X-Hub-Signature-256` header:
- Computed as an HMAC-SHA256 signature using your Meta App Secret.
- Handled with constant-time cryptographic comparison (`subtle.ConstantTimeCompare`) to protect against timing attacks.

### Inbound Media Supported
- **Text messages**
- **Voice notes & Audio files** (converted via Whisper or Google Speech)
- **Photos & Images** (stored in media path and passed with contextual prompt)
- **Documents & PDF files** (saved locally and processed with LLM)

---

## 2. Telegram Bot Webhook

### Webhook Endpoint
- **URL:** `https://your-domain.com/tg`
- **Supported Methods:** `POST` (Telegram updates)

### Setup via Bot API
1. Create a bot using [@BotFather](https://t.me/botfather) and obtain your Bot API Token.
2. Register the webhook and configure a secret token:

```bash
curl -F "url=https://your-domain.com/tg" \
     -F "secret_token=YOUR_OPTIONAL_SECRET_TOKEN" \
     https://api.telegram.org/bot<YOUR_BOT_TOKEN>/setWebhook
```

### Cryptographic Security & Secret Token
If configured, PBX AI validates incoming updates by verifying the `X-Telegram-Bot-Api-Secret-Token` header using constant-time comparison before processing any message payload.

### Inbound Media Supported
- **Text messages** (supports Markdown replies with plain-text fallback)
- **Voice notes** (auto-transcribed via Speech-to-Text)
- **Photos** (highest resolution variant automatically selected and downloaded)
- **Documents** (PDFs, text files, logs)
- **Long message handling:** Responses exceeding 4000 characters are automatically split on paragraph or sentence boundaries and delivered sequentially without truncation.
