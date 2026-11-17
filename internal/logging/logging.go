package logging

import (
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

// InitLogger will init logger globally.
func InitLogger(level slog.Level, show bool) {
	slog.SetDefault(slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		TimeFormat: time.DateTime,
		Level:      level,
		AddSource:  show,
	})))
}
