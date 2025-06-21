package logger

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

func Init() *os.File {
	loggerFile, err := os.Create(filepath.Join(os.TempDir(), "cligram.log"))
	if err != nil {
		fmt.Println(err)
	}

	handler := slog.NewJSONHandler(loggerFile, &slog.HandlerOptions{AddSource: true, Level: slog.LevelDebug})
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return loggerFile
}
