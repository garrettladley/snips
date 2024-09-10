package generatecmd

import (
	"context"
	"log/slog"
)

type Arguments struct {
	FileWriter FileWriterFunc
}

func Run(ctx context.Context, log *slog.Logger, args Arguments) error { return nil }
