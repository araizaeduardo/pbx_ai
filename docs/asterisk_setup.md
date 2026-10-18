# Asterisk Telephony Integration Guide

PBX AI acts as an intelligent FastAGI voice server that Asterisk calls via the standard `AGI()` application in your dialplan. When a call arrives, Asterisk hands control over to PBX AI to handle real-time Voice Activity Detection (VAD), Speech-to-Text (ASR), LLM response synthesis, Text-to-Speech (TTS), and audio playback.

---

## 1. Dialplan Configuration (`extensions.conf`)

Add an extension in your Asterisk dialplan to route incoming calls to the PBX AI FastAGI server:

```ini
[from-internal]
; Route extension 6000 to PBX AI Voice Agent
exten => 6000,1,NoOp(PBX AI Voice Assistant Incoming Call)
 same => n,Answer()
 same => n,Wait(1)
 ; Pass channel parameters or authentication token
 same => n,AGI(agi://127.0.0.1:4573,${CALLERID(num)},token=your_secret_token)
 same => n,Hangup()

; Inbound trunk routing
[from-pstn]
exten => _X.,1,NoOp(Inbound DID call routed to PBX AI)
 same => n,Answer()
 same => n,AGI(agi://127.0.0.1:4573,${CALLERID(num)})
 same => n,Hangup()
```

### FastAGI Call Parameters
PBX AI expects the following arguments from the AGI invocation:
- **Arg 1:** Caller ID / phone number (used as `userId` for conversation state and history).
- **Arg 2 (optional):** Authentication token, passed as `token=<secret>` if `--asterisk-token` is configured.

---

## 2. Dynamic PJSIP Synchronization (`ari.conf` & `pjsip.conf`)

PBX AI can dynamically configure and synchronize SIP endpoints and trunks using the Asterisk REST Interface (ARI).

### `ari.conf`
```ini
[general]
enabled = yes
pretty = yes
allowed_origins = http://localhost:35248

[pbx]
type = user
read_only = no
password = your_asterisk_token
password_format = plain
```

### `http.conf`
```ini
[general]
enabled = yes
bindaddr = 0.0.0.0
bindport = 8088
```

---

## 3. Audio & Transcoding Requirements

PBX AI requires `ffmpeg` and `ffprobe` installed on the host machine or container to transcode between Asterisk telephony formats (8kHz / 16kHz PCM, sln, wav) and cloud provider formats (Ogg Opus, MP3):

```bash
# Verify ffmpeg and ffprobe are available
ffmpeg -version
ffprobe -version
```

### Recommended Audio Codecs in `pjsip.conf`:
```ini
[endpoint_template](!)
type = endpoint
disallow = all
allow = opus
allow = alaw
allow = ulaw
```

---

## 4. DTMF Interaction

PBX AI supports DTMF keypresses (`0-9`, `*`, `#`) during playback. When a caller presses a key:
- The AGI stream detects the ASCII keycode.
- PBX AI can trigger specific actions or branch conversational logic based on numeric input.
