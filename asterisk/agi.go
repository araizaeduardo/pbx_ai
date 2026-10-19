// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package asterisk

import (
	"bufio"
	"crypto/md5"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eja/pbx/db"
	"github.com/eja/pbx/media"
	"github.com/eja/pbx/pbx"
	"github.com/eja/pbx/sys"
)

const recordingTimeout = 30 * 1000

type AgiType struct {
	token     string
	uniqueId  string
	callerId  string
	extension string
	language  string
	request   string
}

func session(conn net.Conn) (err error) {
	const platform = "pbx"
	defer conn.Close()

	reader := bufio.NewReader(conn)
	agi := AgiType{}

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		if strings.HasPrefix(line, "agi_") {
			processHeader(line, &agi)
		} else {
			break
		}
	}

	aiSettings := db.Settings()
	authorized := false
	phone := agi.callerId

	if agi.token == "" {
		if parsedUrl, err := url.Parse(agi.request); err == nil {
			if len(parsedUrl.Path) > 1 {
				agi.token = parsedUrl.Path[1:]
			}
		}
	}

	asteriskToken := aiSettings["asteriskToken"]
	if asteriskToken == "" {
		asteriskToken = sys.Options.AsteriskToken
	}

	asteriskMusic := false
	if sys.Number(aiSettings["asteriskMoh"]) > 0 {
		asteriskMusic = true
	}

	language := aiSettings["language"]
	if agi.language != "en" {
		language = agi.language
	}

	authorizedToken := true
	if asteriskToken != "" {
		if subtle.ConstantTimeCompare([]byte(agi.token), []byte(asteriskToken)) != 1 {
			authorizedToken = false
			slog.Warn("AGI, wrong authorization token", "received", agi.token)
		}
	}

	if authorizedToken {
		if sys.Number(aiSettings["userRestricted"]) == 0 {
			authorized = true
		} else {
			user, err := db.UserGet(phone)
			if err == nil && user != nil {
				authorized = true
				language = user["language"]
			}
		}
	}

	if !authorized {
		return fmt.Errorf("user not authorized")
	}

	vad := sys.Number(aiSettings["asteriskVad"]) > 0
	talking := false
	monitorFile := filepath.Clean(fmt.Sprintf("%s/monitor.%s.%s.wav", sys.Options.MediaPath, phone, agi.uniqueId))

	if agi.extension == "h" {
		_, err = pbx.Chat(platform, phone, "/hangup "+monitorFile, language)
		return
	}

	if _, err = send(conn, reader, "ANSWER"); err != nil {
		return
	}

	mixMonitorTime := time.Now()
	if _, err = send(conn, reader, "EXEC MixMonitor "+monitorFile); err != nil {
		return
	}

	dtmf := ""
	if message, err := pbx.Chat(platform, phone, "/welcome", language); err != nil {
		return err
	} else {
		// Welcome message should always play completely without premature VAD interruption
		_, dtmf, err = play(conn, reader, phone, message, language, false, mixMonitorTime)
		if err != nil {
			return err
		}
		talking = false
	}

	for {
		question := ""
		answer := ""
		hangup := false
		ttsLanguage := language

		if dtmf != "" {
			question = dtmf
			dtmf = ""
		} else {
			if question, err = record(conn, reader, phone, language, vad, talking); err != nil {
				return
			}
		}

		if question != "" {
			if asteriskMusic {
				if _, err = send(conn, reader, "SET MUSIC on"); err != nil {
					return
				}
			}
			if answer, err = pbx.Chat(platform, phone, question, language); err != nil {
				return
			}

			message, tags := pbx.TagsExtract(answer)
			for _, item := range tags {
				lower := strings.ToLower(item)
				switch lower {
				case "close":
					hangup = true
				case "hangup":
					hangup = true
				}
				if len(lower) == 2 {
					ttsLanguage = lower
					if language == "" {
						language = ttsLanguage
					}
				}
				if strings.HasPrefix(lower, "dial:") || strings.HasPrefix(lower, "transfer:") || strings.HasPrefix(lower, "sip:") {
					colonIdx := strings.Index(item, ":")
					dialTarget := strings.TrimSpace(item[colonIdx+1:])

					if !isSafeInternalDestination(dialTarget) {
						slog.Warn("AGI security: blocked unauthorized external transfer attempt", "attempted", dialTarget, "caller", phone)
						refusalMsg := "Por políticas de seguridad, solo es posible transferir a extensiones internas de la empresa."
						if language == "en" {
							refusalMsg = "For security reasons, calls can only be transferred to internal company extensions."
						}
						play(conn, reader, phone, refusalMsg, ttsLanguage, vad, mixMonitorTime)
						continue
					}

					if message != "" {
						if talking, _, err = play(conn, reader, phone, message, ttsLanguage, vad, mixMonitorTime); err != nil {
							return
						}
					}
					message = ""
					if !strings.Contains(dialTarget, "/") {
						dialTarget = fmt.Sprintf("Local/%s@from-ai-transfer", dialTarget)
					}
					if _, err = send(conn, reader, fmt.Sprintf("EXEC DIAL %s", dialTarget)); err != nil {
						return
					}
					hangup = true
				}
			}
			message, err = pbx.TagsProcess(platform, language, phone, message, tags)

			if message != "" {
				if talking, dtmf, err = play(conn, reader, phone, message, ttsLanguage, vad, mixMonitorTime); err != nil {
					return
				}
			}

			if hangup {
				_, err = send(conn, reader, "HANGUP")
				return
			}
		} else {
			// No speech was detected or silence hallucination was filtered out
			talking = false
		}
	}
}

func isSafeInternalDestination(target string) bool {
	clean := target
	if idx := strings.Index(clean, "/"); idx != -1 {
		clean = clean[idx+1:]
	}
	if idx := strings.Index(clean, "@"); idx != -1 {
		clean = clean[:idx]
	}
	clean = strings.TrimSpace(clean)

	if len(clean) < 2 || len(clean) > 4 {
		return false
	}
	for _, c := range clean {
		if c < '0' || c > '9' {
			return false
		}
	}
	if clean == "911" || clean == "112" || clean == "999" || clean == "000" {
		return false
	}
	return true
}

func now() int64 {
	return time.Now().Unix()
}

func record(conn net.Conn, reader *bufio.Reader, phone, language string, vad, talking bool) (string, error) {
	fileName := ""
	silence := 2
	if !talking {
		if _, err := send(conn, reader, `STREAM FILE beep ""`); err != nil {
			return "", err
		}
	}

	asteriskFileName := filepath.Clean(fmt.Sprintf("%s/record.%s.%d", sys.Options.MediaPath, phone, now()))
	fileName = asteriskFileName + ".wav16"

	if !talking {
		if _, err := send(conn, reader, "EXEC WaitForNoise 1"); err != nil {
			return "", err
		}
	}
	if msg, err := send(conn, reader, fmt.Sprintf("RECORD FILE %s wav16 # %d 1 s=%d", asteriskFileName, recordingTimeout, silence)); err != nil {
		return msg, err
	}

	return pbx.ASR(fileName, language)
}

func play(conn net.Conn, reader *bufio.Reader, phone string, message string, language string, vad bool, mixMonitorTime time.Time) (talking bool, dtmf string, err error) {
	fileOutputName := filepath.Clean(fmt.Sprintf("%s/tts.%x.wav", sys.Options.Cache, md5.Sum([]byte(message))))
	fileOutputTmp := filepath.Clean(fmt.Sprintf("%s/%s.tts.%d", sys.Options.MediaPath, phone, now()))
	if _, err = os.Stat(fileOutputName); err != nil {
		if err = pbx.TTS(message, language, fileOutputTmp); err != nil {
			return
		}
		if err = media.MpegAudioAsterisk(fileOutputTmp, fileOutputName); err != nil {
			return
		}
	} else {
		slog.Debug("AGI, using tts transcoded cache", "phone", phone, "message", message)
	}

	if err != nil {
		return
	}
	asteriskFileName := strings.TrimSuffix(fileOutputName, ".wav")

	if vad {
		probeInput, err := media.ProbeAudio(fileOutputName)
		if err != nil {
			return false, "", err
		}
		slog.Debug("AGI play", "input", probeInput)
		playTimeFile := sys.Float(probeInput["duration"])
		playTimeStart := time.Now().UnixNano()
		if msg, err := send(conn, reader, fmt.Sprintf("EXEC BackgroundDetect %s,800,400", asteriskFileName)); err != nil {
			return false, "", err
		} else {
			dtmf = dtmfExtract(msg)
			if dtmf != "" {
				for {
					if msg, err := send(conn, reader, fmt.Sprintf("EXEC WaitExten 1.5")); err != nil {
						return false, "", err
					} else {
						val := dtmfExtract(msg)
						if val == "" {
							break
						} else {
							dtmf += val
						}
					}
				}
			}
		}
		playTimeStop := time.Now().UnixNano()
		playTimeDiff := sys.Float(playTimeStop-playTimeStart) / 1e9
		if playTimeDiff < playTimeFile {
			slog.Debug("AGI, talking detected")
			talking = true
		}
	} else {
		_, err = send(conn, reader, fmt.Sprintf("CONTROL STREAM FILE %s # 1000 6 4 5 0", asteriskFileName))
	}

	return
}

func dtmfExtract(msg string) string {
	if len(msg) < 12 {
		return ""
	}
	fields := strings.Fields(msg[11:])
	if len(fields) == 0 {
		return ""
	}
	val := sys.Number(fields[0])
	if val > 0 {
		return string(rune(val))
	}
	return ""
}

func processHeader(line string, agi *AgiType) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	switch key {
	case "agi_uniqueid":
		agi.uniqueId = value
	case "agi_callerid":
		agi.callerId = value
	case "agi_language":
		agi.language = value
	case "agi_extension":
		agi.extension = value
	case "agi_request":
		agi.request = value
	case "agi_arg_1":
		agi.token = value
	}
}

func send(conn net.Conn, reader *bufio.Reader, tx string) (rx string, err error) {
	if _, err = conn.Write([]byte(tx + "\n")); err != nil {
		return
	}

	if rx, err = reader.ReadString('\n'); err != nil {
		return
	}

	slog.Debug("AGI", "tx", tx, "rx", rx)

	if strings.HasPrefix(rx, "200") {
		return strings.TrimSpace(rx), nil
	} else {
		return rx, fmt.Errorf("AGI rx/tx error")
	}
}

func Start() error {
	listener, err := net.Listen("tcp", sys.Options.AsteriskAgi)
	if err != nil {
		return err
	}
	defer listener.Close()

	slog.Info("AGI ready", "address", sys.Options.AsteriskAgi)
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		go func(conn net.Conn) {
			if err := session(conn); err != nil {
				slog.Warn("AGI error", "error", err)
			}
		}(conn)
	}
}
