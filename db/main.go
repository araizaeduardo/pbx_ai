// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package db

import (
	"fmt"
	"sync"

	"github.com/eja/pbx/sys"
)

func UserGet(uuid string) (tibulaTypeRow, error) {
	return tibulaRow(`SELECT * FROM aiUsers WHERE uuid = ? AND (expiration IS NULL OR expiration = "" OR expiration > ?) LIMIT 1`,
		uuid, tibulaNow())
}

// UserGetWithPassword returns the aiUsers row matching uuid and password.
// Passwords are stored as bcrypt (salted) hashes, so they must be verified
// with PasswordCheck instead of comparing a freshly generated hash in SQL.
func UserGetWithPassword(uuid, password string) (tibulaTypeRow, error) {
	row, err := tibulaRow(`SELECT * FROM aiUsers WHERE uuid = ? AND (expiration IS NULL OR expiration = "" OR expiration > ?) LIMIT 1`,
		uuid, tibulaNow())
	if err != nil {
		return nil, err
	}
	if len(row) == 0 || row["password"] == "" || !tibulaSession.PasswordCheck(password, row["password"]) {
		return nil, fmt.Errorf("invalid credentials")
	}
	return row, nil
}

// AdminCheck verifies credentials against the Tibula administration users (ejaUsers).
func AdminCheck(username, password string) bool {
	if username == "" || password == "" {
		return false
	}
	db, err := tibulaConnection()
	if err != nil {
		return false
	}
	defer db.Close()
	return len(db.UserGetAllByUserAndPass(username, password)) > 0
}

var userUpdatableFields = map[string]bool{
	"welcome":  true,
	"language": true,
	"audio":    true,
	"name":     true,
	"note":     true,
}

func UserUpdate(uuid string, field string, value string) (err error) {
	if !userUpdatableFields[field] {
		return fmt.Errorf("field %q cannot be updated", field)
	}
	query := fmt.Sprintf("UPDATE aiUsers SET %s = ? WHERE uuid = ?", field)
	_, err = tibulaRun(query, value, uuid)
	return
}

func SystemPrompt(platform string) (tibulaTypeRows, error) {
	return tibulaRows("SELECT label,prompt FROM aiPrompts WHERE active > 0 AND (platform='' OR platform='all' OR platform=?) ORDER BY power ASC", platform)
}

func ChatAction(platform, action, language string) (function, response string) {
	values, err := tibulaRow(`SELECT function, response FROM aiActions WHERE 
			active>0 AND 
			action=? AND 
			(language=? OR language='') AND 
			(platform='' OR platform='all' OR platform=?) 
			ORDER by language DESC LIMIT 1`,
		action, language, platform)
	if err == nil {
		function = values["function"]
		response = values["response"]
	}
	return
}

func Settings() (values tibulaTypeRow) {
	values, _ = tibulaRow(`SELECT * FROM aiSettings WHERE ejaId=1`)
	return
}

type LogEntry struct {
	Source    string `json:"source"`
	Message   string `json:"message"`
	Channel   string `json:"channel"`
	Tokens    int    `json:"tokens"`
	LatencyMs int64  `json:"latency_ms"`
	Model     string `json:"model"`
	Error     string `json:"error"`
}

var migratedAiLogs bool
var migrateMutex sync.Mutex

func ensureAiLogsColumns() {
	migrateMutex.Lock()
	defer migrateMutex.Unlock()
	if migratedAiLogs {
		return
	}
	existingCols := make(map[string]bool)
	if rows, err := tibulaRows("PRAGMA table_info(aiLogs)"); err == nil {
		for _, r := range rows {
			existingCols[r["name"]] = true
		}
	}
	cols := map[string]string{
		"channel":    "ALTER TABLE aiLogs ADD COLUMN channel TEXT",
		"tokens":     "ALTER TABLE aiLogs ADD COLUMN tokens INTEGER DEFAULT 0",
		"latency_ms": "ALTER TABLE aiLogs ADD COLUMN latency_ms INTEGER DEFAULT 0",
		"model":      "ALTER TABLE aiLogs ADD COLUMN model TEXT",
		"error":      "ALTER TABLE aiLogs ADD COLUMN error TEXT",
	}
	for name, sql := range cols {
		if !existingCols[name] {
			_, _ = tibulaRun(sql)
		}
	}
	migratedAiLogs = true
}

func LogEntryWrite(entry LogEntry) error {
	settings := Settings()
	if tibulaNumber(settings["logs"]) > 0 {
		ensureAiLogsColumns()
		if _, err := tibulaRun(
			`INSERT INTO aiLogs (ejaOwner, ejaLog, source, message, channel, tokens, latency_ms, model, error) VALUES (1,?,?,?,?,?,?,?,?)`,
			tibulaNow(), entry.Source, entry.Message, entry.Channel, entry.Tokens, entry.LatencyMs, entry.Model, entry.Error,
		); err != nil {
			// Fallback to basic columns if needed
			if _, err2 := tibulaRun(`INSERT INTO aiLogs (ejaOwner, ejaLog, source, message) VALUES (1,?,?,?)`, tibulaNow(), entry.Source, entry.Message); err2 != nil {
				return err2
			}
			return err
		}
	}
	return nil
}

func Log(source, message string) error {
	return LogEntryWrite(LogEntry{
		Source:  source,
		Message: message,
	})
}

type DbDashboardStats struct {
	TotalRequests int64            `json:"total_requests"`
	TotalTokens   int64            `json:"total_tokens"`
	AvgLatencyMs  int64            `json:"avg_latency_ms"`
	TotalErrors   int64            `json:"total_errors"`
	ChannelCounts map[string]int64 `json:"channel_counts"`
}

func GetDashboardStats() (DbDashboardStats, error) {
	ensureAiLogsColumns()
	var stats DbDashboardStats
	stats.ChannelCounts = make(map[string]int64)

	total, _ := tibulaValue("SELECT COUNT(*) FROM aiLogs")
	stats.TotalRequests = int64(tibulaNumber(total))

	toks, _ := tibulaValue("SELECT COALESCE(SUM(tokens), 0) FROM aiLogs")
	stats.TotalTokens = int64(tibulaNumber(toks))

	avgLat, _ := tibulaValue("SELECT COALESCE(AVG(latency_ms), 0) FROM aiLogs WHERE latency_ms > 0")
	stats.AvgLatencyMs = int64(tibulaFloat(avgLat))

	errCount, _ := tibulaValue("SELECT COUNT(*) FROM aiLogs WHERE error IS NOT NULL AND error != ''")
	stats.TotalErrors = int64(tibulaNumber(errCount))

	rows, err := tibulaRows("SELECT COALESCE(channel, 'unknown') as ch, COUNT(*) as cnt FROM aiLogs GROUP BY ch")
	if err == nil {
		for _, r := range rows {
			ch := r["ch"]
			if ch == "" {
				ch = "unknown"
			}
			stats.ChannelCounts[ch] = int64(tibulaNumber(r["cnt"]))
		}
	}

	return stats, nil
}

func GetRecentLogs(limit int, channel, search string) (tibulaTypeRows, error) {
	ensureAiLogsColumns()
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	whereClause := "1=1"
	var args []any
	if channel != "" && channel != "all" {
		whereClause += " AND channel = ?"
		args = append(args, channel)
	}
	if search != "" {
		whereClause += " AND (source LIKE ? OR message LIKE ? OR error LIKE ?)"
		s := "%" + search + "%"
		args = append(args, s, s, s)
	}

	query := fmt.Sprintf("SELECT ejaId, ejaLog, source, message, COALESCE(channel, 'chat') as channel, COALESCE(tokens, 0) as tokens, COALESCE(latency_ms, 0) as latency_ms, COALESCE(model, '') as model, COALESCE(error, '') as error FROM aiLogs WHERE %s ORDER BY ejaId DESC LIMIT %d", whereClause, limit)
	return tibulaRows(query, args...)
}

var (
	migratedAiConversations bool
	convMigrateMutex        sync.Mutex
)

func ensureAiConversationsTable() {
	convMigrateMutex.Lock()
	defer convMigrateMutex.Unlock()
	if migratedAiConversations {
		return
	}
	query := `CREATE TABLE IF NOT EXISTS aiConversations (
		ejaId INTEGER PRIMARY KEY AUTOINCREMENT,
		ejaOwner INTEGER DEFAULT 1,
		ejaLog DATETIME,
		userId TEXT,
		role TEXT,
		content TEXT
	)`
	_, _ = tibulaRun(query)
	_, _ = tibulaRun(`CREATE INDEX IF NOT EXISTS idx_aiConversations_user ON aiConversations (userId, ejaId)`)
	migratedAiConversations = true
}

func ConversationSave(userId, role, content string) error {
	ensureAiConversationsTable()
	_, err := tibulaRun(
		`INSERT INTO aiConversations (ejaOwner, ejaLog, userId, role, content) VALUES (1,?,?,?,?)`,
		tibulaNow(), userId, role, content,
	)
	return err
}

func ConversationLoad(userId string, limit int) ([]sys.TypeChatMessage, error) {
	ensureAiConversationsTable()
	if limit <= 0 {
		limit = 50
	}
	rows, err := tibulaRows(
		`SELECT role, content FROM aiConversations WHERE userId = ? ORDER BY ejaId DESC LIMIT ?`,
		userId, limit,
	)
	if err != nil {
		return nil, err
	}
	result := make([]sys.TypeChatMessage, len(rows))
	for i, r := range rows {
		result[len(rows)-1-i] = sys.TypeChatMessage{
			Role:    r["role"],
			Content: r["content"],
		}
	}
	return result, nil
}

func ConversationReset(userId string) error {
	ensureAiConversationsTable()
	_, err := tibulaRun(`DELETE FROM aiConversations WHERE userId = ?`, userId)
	return err
}

func DefaultLanguage() (string, error) {
	return tibulaValue("SELECT language FROM aiSettings WHERE ejaId=1")
}

func Translate(label, language string) (string, error) {
	return tibulaValue("SELECT translation FROM aiTranslations WHERE label=? AND language=? LIMIT 1", label, language)
}

func LanguageCodeToLocale(language string) (string, error) {
	return tibulaValue("SELECT locale FROM aiLanguages WHERE code = ? LIMIT 1", language)
}

func LanguageCodeToInternal(language string) (string, error) {
	return tibulaValue("SELECT internal FROM aiLanguages WHERE code = ? LIMIT 1", language)
}
