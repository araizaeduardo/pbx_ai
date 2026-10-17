# Tareas PBX AI

Leyenda: `[ ]` pendiente · `[~]` en progreso · `[x]` hecho
Prioridad: 🔴 crítica · 🟠 alta · 🟡 media · 🟢 baja

---

## Fase 1 — Bugs críticos y de alta severidad

- [x] 🔴 **T-01** Proteger `history`, `historyTime`, `historyThread` con `sync.RWMutex` (`pbx/chat.go`, `pbx/plugins.go`)
- [x] 🔴 **T-02** Limitar historial por usuario (máx. N mensajes) y añadir expiración/limpieza (`pbx/chat.go`)
- [x] 🟠 **T-03** Corregir `UserUpdate`: `WHERE id = ?` → `WHERE uuid = ?` (`db/main.go`)
- [x] 🟠 **T-04** Corregir `FilterLanguage`: el regex espera `[xx]` pero los tags ya no llevan corchetes (`pbx/tags.go`)
- [x] 🟠 **T-05** `Ntfy`: devolver error real cuando el status HTTP != 200 (`sys/ntfy.go`)
- [x] 🟠 **T-06** Validar `StatusCode` en OpenAI STT y TTS; no escribir JSON de error como audio (`openai/stt.go`, `openai/tts.go`)
- [x] 🟠 **T-07** Validar `len(results) == 0` en Google ASR antes de `results[0]` (`google/asr.go`)

## Fase 2 — Bugs de severidad media

- [x] 🟡 **T-08** `pbx.Audio`: se mantiene `"chat"` (Meta y Telegram usan los prompts de esa plataforma); `TagsProcess` ahora usa `chatPlatform` (`pbx/main.go`)
- [x] 🟡 **T-09** `chatRestart` debe enviar `/reset` al servidor antes de recargar (`web/assets/script.js`)
- [x] 🟡 **T-10** Resetear altura del `textarea` tras enviar mensaje (`web/assets/script.js`)
- [x] 🟡 **T-11** Añadir `Timeout` a todos los `http.Client` (`pbx/llm.go`, `asterisk/ari.go`, `meta/main.go`, `openai/*`, `google/*`, `telegram/main.go`)
- [x] 🟡 **T-12** Comparar claves de ffprobe en snake_case: `codec_name`, `sample_rate`, `channel_layout` (`meta/main.go`)
- [x] 🟡 **T-13** Verificar status HTTP en `telegram.SendText` y `SendAudio`
- [x] 🟡 **T-14** Corregir `slog.Debug("ffmpeg", args)` (falta clave/valor) (`media/main.go`)

## Fase 3 — Calidad de código (baja severidad)

- [ ] 🟢 **T-15** Validar longitud en `dtmfExtract` (`msg[11:]`) (`asterisk/agi.go`)
- [ ] 🟢 **T-16** Revisar `set_var=set_var=...` duplicado en ARI (`asterisk/ari.go`)
- [ ] 🟢 **T-17** `fmt.Errorf("Cannot create media folder", err)` sin verbo (`sys/main.go`)
- [x] 🟢 **T-18** Mensaje de log `"mail error"` en plugin `ntfy` (`pbx/plugins.go`)
- [ ] 🟢 **T-19** `send()` en AGI crea un `bufio.Reader` nuevo por llamada; reutilizar el de la sesión (`asterisk/agi.go`)
- [ ] 🟢 **T-20** Evitar `regexp.MustCompile` en cada llamada; moverlo a variable global (`pbx/tags.go`)
- [ ] 🟢 **T-21** Reutilizar `http.Client` compartido en vez de crear uno por petición

## Fase 4 — Seguridad

- [ ] 🟠 **T-22** Restringir CORS `Access-Control-Allow-Origin: *` en `/chat` (configurable) (`web/chat.go`)
- [ ] 🟠 **T-23** Validar firma `X-Hub-Signature-256` en webhook de Meta y secret token en Telegram (`web/meta.go`, `web/telegram.go`)
- [ ] 🟡 **T-24** Comparar token AGI con `subtle.ConstantTimeCompare` (`asterisk/agi.go`)
- [ ] 🟡 **T-25** No exponer la API key de Google en la URL (usar header) y evitar loguearla (`google/*`)
- [ ] 🟡 **T-26** Limitar tamaño del body JSON en `/chat` con `http.MaxBytesReader`
- [ ] 🟡 **T-27** Rate limiting por usuario/IP en chat y webhooks
- [x] 🟡 **T-28** Whitelist de campos en `UserUpdate` (evitar SQL por nombre de columna)
- [ ] 🟢 **T-29** Revisar hash de contraseñas de Tibula (`tibulaPassword`) y política de expiración
- [ ] 🟢 **T-30** Sanitizar HTML renderizado con `marked` (usar DOMPurify) en el chat web

## Fase 5 — Pruebas, CI y mantenimiento

- [ ] 🟠 **T-31** Crear tests unitarios: `TagsExtract`, `FilterLanguage`, `MarkdownToText`, `dtmfExtract`
- [ ] 🟡 **T-32** Tests con mock HTTP para `LLM()` (tool calls, errores, max iteraciones)
- [ ] 🟡 **T-33** Workflow de GitHub Actions: `go vet`, `go test -race`, `staticcheck`, build
- [x] 🟡 **T-34** Instalar Go en el entorno local y verificar que compila (`go` no está en el PATH)
- [ ] 🟢 **T-35** Añadir `.golangci.yml` y `make lint`
- [ ] 🟢 **T-36** Actualizar dependencias (`go get -u`, `go mod tidy`) y revisar `govulncheck`

## Fase 6 — Mejoras funcionales

- [ ] 🟠 **T-37** Persistir historial de conversaciones en BD (sobrevive a reinicios)
- [ ] 🟡 **T-38** Reintentos con backoff ante 429/5xx del LLM
- [ ] 🟡 **T-39** Streaming de respuestas del LLM en el chat web (SSE)
- [ ] 🟡 **T-40** Soporte de imágenes/documentos entrantes en WhatsApp y Telegram
- [ ] 🟡 **T-41** Compartir caché de clientes y herramientas MCP con reconexión automática
- [ ] 🟡 **T-42** Dividir mensajes largos según límite de Telegram (4096) y WhatsApp
- [ ] 🟢 **T-43** Formato Markdown/HTML en respuestas de Telegram
- [ ] 🟢 **T-44** Permitir seleccionar voz y modelo TTS por idioma o usuario
- [ ] 🟢 **T-45** Soporte de otros proveedores ASR/TTS (ElevenLabs, Azure, Deepgram)

## Fase 7 — Dashboard y observabilidad

- [ ] 🟠 **T-46** Ampliar `aiLogs`: canal, tokens, latencia, modelo, error
- [ ] 🟠 **T-47** Endpoint `/metrics` en formato Prometheus (peticiones, latencia, errores, tokens)
- [ ] 🟡 **T-48** Dashboard web: mensajes/día, llamadas, usuarios activos, costos estimados
- [ ] 🟡 **T-49** Vista de conversaciones en vivo / historial por usuario en el panel
- [ ] 🟡 **T-50** Alertas (ntfy/mail) ante errores repetidos o caídas de proveedores
- [ ] 🟢 **T-51** Endpoint `/health` para monitoreo y Docker
- [ ] 🟢 **T-52** Logging estructurado con niveles configurables por flag

## Fase 8 — Despliegue y documentación

- [ ] 🟡 **T-53** `Dockerfile` y `docker-compose.yml` (pbx + Asterisk + DB)
- [ ] 🟡 **T-54** Documentar variables, flags y ejemplo de `pbx.json`
- [ ] 🟡 **T-55** Guía de configuración de Asterisk (dialplan, PJSIP, AGI)
- [ ] 🟢 **T-56** Guía de webhooks de Meta y Telegram
- [ ] 🟢 **T-57** Documentar arquitectura y flujo de mensajes (diagrama)
- [ ] 🟢 **T-58** `CHANGELOG.md` y política de versionado

---

## Resumen

| Fase | Tareas | Foco |
|---|---|---|
| 1 | T-01 a T-07 | Bugs críticos |
| 2 | T-08 a T-14 | Bugs medios |
| 3 | T-15 a T-21 | Calidad |
| 4 | T-22 a T-30 | Seguridad |
| 5 | T-31 a T-36 | Pruebas y CI |
| 6 | T-37 a T-45 | Funcionalidad |
| 7 | T-46 a T-52 | Dashboard y observabilidad |
| 8 | T-53 a T-58 | Despliegue y docs |

**Orden sugerido:** Fase 1 → Fase 5 (T-34, T-31, T-33) → Fase 2 → Fase 4 → Fase 7 → resto.
