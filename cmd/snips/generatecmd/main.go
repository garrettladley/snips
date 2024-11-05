package generatecmd

import (
	"context"
	_ "embed"
	"log/slog"

	_ "net/http/pprof"
)

type Arguments struct {
	FileName          string
	FileWriter        FileWriterFunc
	Path              string
	Watch             bool
	Style             string
	TabWidth          int
	Lines             bool
	LinesTable        bool
	BaseLine          int
	LinkableLines     bool
	WorkerCount       int
	KeepOrphanedFiles bool
	Lazy              bool
}

func Run(ctx context.Context, log *slog.Logger, args Arguments) (err error) {
	return NewGenerate(log, args).Run(ctx)
}
