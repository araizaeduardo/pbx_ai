# PBX: AI-Powered Unified Communications Bridge

A comprehensive communication middleware designed to integrate telephony and messaging platforms with Large Language Model (LLM) intelligence. By bridging **Asterisk**, **WhatsApp**, and **Telegram**, the platform delivers interactive, AI-driven experiences for both voice and text interactions. Built upon the **Tibula RDBMS** framework, PBX provides granular control over user authentication, system-wide prompt engineering, and multi-language communication workflows.

## Key Capabilities

*   **Unified Multi-Platform Integration:** Centralized management of voice and messaging traffic across **Asterisk (AGI/ARI)**, **WhatsApp**, and **Telegram** endpoints.
*   **Advanced Conversational AI:**
    *   **LLM Interoperability:** Support for OpenAI-compatible APIs with **Model Context Protocol (MCP)** integration, allowing AI models to leverage custom internal tools.
    *   **Context Management:** Sophisticated stateful chat history tracking with automated session timeout handling.
    *   **Command Logic:** A robust system for parsing custom commands (e.g., `/reset`, `/mail`, `/ntfy`) to trigger asynchronous system actions.
*   **Media Processing Engine:**
    *   **Intelligent ASR/TTS:** Seamless integration with OpenAI (Whisper/TTS) or Google Speech services for high-fidelity audio conversion.
    *   **VoIP Optimization:** Real-time audio transcoding and Voice Activity Detection (VAD) specifically engineered for stable Asterisk performance.
*   **Scalable Architecture:** A modular plugin-based design facilitating custom chat actions and a database-driven configuration system for rapid deployment and maintenance.

## System Prerequisites

*   **Dependencies:** `ffmpeg` and `ffprobe` (for real-time media transcoding and metadata analysis).
*   **Database Management:** A RDBMS compatible with the Tibula framework.
*   **API Infrastructure:**
    *   **LLM Provider:** API access to OpenAI or an OpenAI-compatible endpoint.
    *   **Speech Services:** Credentials for OpenAI (Whisper/TTS) or Google Cloud Speech APIs.
    *   **Communication APIs:** Valid credentials for the Telegram Bot API or Meta/WhatsApp Business API.
*   **Telephony Environment:** A configured Asterisk server with PJSIP capability.

## Quick Start Guide

### 1. Installation
Clone the repository and compile the binary:

```bash
git clone https://github.com/eja/pbx
cd pbx
make
```

### 2. Initial Configuration
Execute the integrated setup wizard to configure database parameters, API tokens, and platform integration endpoints:

```bash
./build/pbx --wizard
```

### 3. Service Deployment
Initiate the service to deploy the web interface, the telephony FastAGI server, and the webhook listeners:

```bash
./build/pbx --start --db-name pbx.db --chat --web-port 35248
```

---

## Operations & Telephony Dashboard

PBX AI includes a built-in dark-mode operations dashboard accessible at:
```
http://localhost:35248/dashboard
```

- **Live Session Inspector:** View in-memory user conversations in real-time.
- **Audit Logs:** Search and filter historical interactions (`aiLogs`) across Asterisk, WhatsApp, Telegram, and Web.
- **Channel Breakdown & KPIs:** Total requests, token consumption, estimated costs, and average latency.
- **Provider Status:** Live status check for LLM, Asterisk ARI/AGI, Meta, and Telegram.

---

## Observability & Health

- **Health Check (`GET /health`):** JSON endpoint returning service uptime, database health, and channel states. Suitable for Docker and Kubernetes probes.
- **Prometheus Metrics (`GET /metrics`):** Standard Prometheus metrics scraper exposing request counts, token consumption, latency counters, and active session gauges.

---

## Docker Deployment

Run PBX AI instantly with Docker Compose:

```bash
docker-compose up -d
```

Or build and run using Docker:

```bash
docker build -t pbx_ai .
docker run -d -p 35248:35248 -p 4573:4573 --name pbx_ai pbx_ai
```

---

## Documentation & Guides

- [Configuration & CLI Reference](docs/config.md)
- [System Architecture & Message Flow](docs/architecture.md)
- [Asterisk Telephony Integration Guide](docs/asterisk_setup.md)
- [Meta (WhatsApp) & Telegram Webhooks Guide](docs/webhooks.md)
- [Changelog](CHANGELOG.md)

