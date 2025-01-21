package monitor

import (
	"encoding/json"
	"logmonitor/input"
	"path/filepath"
)

type ZeroLog struct {
	webhook Notifier
	fileName string
}

func NewZeroLog(logFile string, webhook Notifier) *ZeroLog {
	fileName := filepath.Base(logFile)
	return &ZeroLog{fileName: fileName, webhook: webhook}
}

func (m *ZeroLog) Process(lineReader *input.Reader) error {
	line, err := lineReader.ReadString("\n")
	if err != nil {
		return err
	}
	var entry struct {
            Level string    `json:"level"`
        }
        if err := json.Unmarshal([]byte(line), &entry); err != nil {
            return err
        }
	if entry.Level == "error" {
		m.webhook.Send(m.fileName, line)
	}
	return nil
}
