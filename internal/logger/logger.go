package logger

import (
	"fmt"
	"os"

	"artnet2mqtt/internal/config"
	"github.com/sirupsen/logrus"
)

type Log struct {
	*logrus.Entry
}

// NewLogger конструктор.
func NewLogger(cfg config.LogConf) (*Log, error) {
	log := logrus.New()

	log.SetOutput(os.Stdout)

	log.Formatter = &logrus.TextFormatter{
		TimestampFormat:  "2006-01-02 15:04:05.0000",
		DisableColors:    false,
		ForceColors:      true,
		FullTimestamp:    true,
		QuoteEmptyFields: true,
	}

	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("logger. Error in settings (level: %s): %w", cfg.Level, err)
	}
	log.SetLevel(level)
	// Disable concurrency mutex as we use Stdout.
	log.SetNoLock()
	log.Debug("set level: ", level)

	return &Log{Entry: log.WithFields(nil)}, nil
}

// With will add the fields to the formatted log entry.
func (l *Log) With(fields Fields) *Log {
	return &Log{Entry: l.WithFields(logrus.Fields(fields))}
}

func (l *Log) GetLevel() string {
	return l.Logger.Level.String()
}

// Fields are a representation of formatted log fields.
type Fields map[string]interface{}

// Logger интерфейс для регистратора.
type Logger interface {
	// GetLevel возвращает текущий установленный уровень логирования.
	GetLevel() string
	With(fields Fields) *Log
}