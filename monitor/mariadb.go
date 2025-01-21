package monitor

import (
	"fmt"
	"path/filepath"
	"strings"

	"logmonitor/input"
)

type MariaDB struct {
	webhook Notifier
	fileName string
}

func NewMariaDB(logFile string, webhook Notifier) *MariaDB {
	fileName := filepath.Base(logFile)
	return &MariaDB{fileName: fileName, webhook: webhook}
}

func (m *MariaDB) skipToLine(reader *input.Reader, keyword string, dropLine bool) error {
	for {
		line, err := reader.PeekString("\n")
		if err != nil {
			return err
		}
		line = strings.TrimSpace(line)
		keyword = strings.TrimSpace(keyword)
		fmt.Printf("line1--:'%s' '%s' '%t'\n", line, keyword, strings.HasPrefix(line, keyword))
		if strings.HasPrefix(line, keyword) {
			fmt.Printf("line2--:%s\n", line)
			return nil
		}
		if dropLine {
			reader.ReadString("")
		}
	}
}

func (m *MariaDB) Process(reader *input.Reader) error {
	line, err := reader.PeekString(";")
	if err != nil {
		return err
	}
	if strings.Contains(line, "SET timestamp=") {
		return nil
	}
	line, err = reader.ReadString("")
	if err != nil {
		return err
	}
	m.webhook.Send(m.fileName, line)
	return nil
}
