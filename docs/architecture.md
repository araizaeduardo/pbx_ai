# PBX AI — System Architecture & Data Flow

This document details the architectural layers, message routing pipelines, telephony lifecycles, and observability systems implemented across PBX AI.

---

## 1. High-Level Architecture

```mermaid
graph TD
    subgraph "External Channels"
        PSTN["PSTN / SIP Callers"]
        WA["WhatsApp (Meta)"]
        TG["Telegram"]
        WEB["Web Browser Users"]
    end

    subgraph "Edge Ingestion & Security"
        AGI["FastAGI Server (:4573)<br/>Constant-Time Auth"]
        RL["Sliding-Window Rate Limiter<br/>HMAC-SHA256 Signature Validator"]
        CORS["CORS & Body Size Limiter<br/>(1MB JSON, 50MB Audio)"]
    end

    subgraph "PBX Core Engine"
        ROUTER["Platform Router (pbx.Text / pbx.Audio)"]
        HIST["Context State Manager<br/>(In-Memory + SQLite/MySQL aiConversations)"]
        TAGS["Action & Tag Parser ([en], /welcome, /reset)"]
        RETRY["LLM Engine with Exponential Backoff<br/>(429 / 5xx Retries)"]
        MCP["Model Context Protocol (MCP)<br/>Client Cache & Auto-Reconnect"]
    end

    subgraph "Audio & Speech Layer"
        FFMPEG["ffmpeg & ffprobe Transcoding Engine"]
        ASR["Speech-to-Text (Whisper / Google)"]
        TTS["Text-to-Speech (OpenAI / Google / ElevenLabs)"]
    end

    subgraph "Observability & Storage"
        DB[("Database (aiLogs, aiConversations, aiSettings)")]
        METRICS["Prometheus Exporter (/metrics)"]
        HEALTH["Health Monitor (/health)"]
        DASH["Telephony & Ops Dashboard (/dashboard)"]
        ALERTS["Provider Failure Alert Manager (ntfy/push)"]
    end

    PSTN --> AGI
    WA --> RL
    TG --> RL
    WEB --> CORS

    AGI --> ROUTER
    RL --> ROUTER
    CORS --> ROUTER

    ROUTER --> HIST
    HIST --> TAGS
    TAGS --> RETRY
    RETRY --> MCP

    ROUTER --> FFMPEG
    FFMPEG --> ASR
    RETRY --> TTS
    TTS --> FFMPEG

    ROUTER --> DB
    ROUTER --> METRICS
    ROUTER --> ALERTS
    DB --> DASH
    METRICS --> DASH
    HEALTH --> DASH
```

---

## 2. Telephony Voice Call Flow (Asterisk FastAGI)

```mermaid
sequenceDiagram
    autonumber
    actor Caller
    participant Ast as Asterisk PBX
    participant FastAGI as PBX AI FastAGI (:4573)
    participant Core as PBX Core
    participant ASR as Whisper / Google ASR
    participant LLM as LLM Engine
    participant TTS as ElevenLabs / OpenAI TTS

    Caller->>Ast: Dials Voice DID / Extension
    Ast->>FastAGI: AGI Connect (CallerID, Auth Token)
    FastAGI->>FastAGI: Constant-Time Token Verify
    FastAGI->>Core: Fetch /welcome prompt
    Core->>FastAGI: Welcome speech text
    FastAGI->>Ast: STREAM FILE (Playback welcome audio)
    
    loop Conversational Turn
        Ast->>FastAGI: RECORD FILE (Capture caller audio)
        FastAGI->>ASR: Transcode & Speech-to-Text
        ASR-->>FastAGI: Return Transcript
        FastAGI->>Core: Process Turn (Restore DB History)
        Core->>LLM: Multi-turn prompt + System Prompt
        LLM-->>Core: Synthesized Answer
        Core->>TTS: Text-to-Speech (Cached WAV)
        TTS-->>FastAGI: Audio file ready
        FastAGI->>Ast: STREAM FILE (Playback answer to caller)
        Note over Ast,FastAGI: Listens for DTMF interruption (0-9, *, #)
    end

    Caller->>Ast: Hangup call
    Ast->>FastAGI: HANGUP
    FastAGI->>Core: Close Session & Log Call Metrics
```

---

## 3. Webhook Messaging Pipeline (WhatsApp / Telegram / Web)

```mermaid
sequenceDiagram
    autonumber
    participant Client as User (WhatsApp / Telegram / Web)
    participant Web as PBX AI Web Server (:35248)
    participant Sec as Security & Rate Limiter
    participant Core as PBX Core
    participant LLM as LLM Engine with Backoff
    participant Split as Message Splitter

    Client->>Web: Incoming Message POST
    Web->>Sec: Check sliding-window rate limit
    Sec->>Sec: Verify HMAC-SHA256 / Secret Token
    Sec-->>Web: Request Authenticated
    Web->>Core: Route Message (Load past conversation from DB)
    Core->>LLM: Send Context + User Input
    
    alt Transient 429 or 5xx Provider Error
        LLM->>LLM: Exponential Backoff Retry (500ms, 1s, 2s)
    end
    
    LLM-->>Core: Response generated
    Core->>Core: Persist user & assistant turns to DB
    Core->>Split: Check length against channel limit (4096)
    
    alt Exceeds 4000 characters
        Split-->>Web: Sequential message chunks (split on \n\n or . )
        loop For each chunk
            Web->>Client: Deliver chunk in order (Markdown / Plaintext fallback)
        end
    else Normal message
        Web->>Client: Deliver single message / SSE stream
    end
```

---

## 4. Observability & Metrics Architecture

- **Prometheus Scraper (`GET /metrics`):**
  Exposes gauges and counters for active sessions, total requests by channel, tokens consumed by model, and request duration.
- **Health Check (`GET /health`):**
  Provides structured JSON health statuses for container orchestration and uptime probes.
- **Operations Dashboard (`GET /dashboard`):**
  Real-time dark-mode control center rendering live sessions, audit logs (`aiLogs`), and provider statuses.
