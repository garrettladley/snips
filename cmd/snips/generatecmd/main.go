package generatecmd

import (
	"context"
	_ "embed"
	"log/slog"

	_ "net/http/pprof"
)

type Arguments struct {
	FileName              string
	FileWriter            FileWriterFunc
	Path                  string
	Watch                 bool
	Style                 string
	Prefix                string
	Styles                bool
	AllStyles             bool
	HTMLOnly              bool
	InlineStyles          bool
	TabWidth              int
	Lines                 bool
	LinesTable            bool
	LinesStyle            string
	Highlight             string
	HighlightStyle        string
	BaseLine              int
	PreventSurroundingPre bool
	LinkableLines         bool
	WorkerCount           int
	KeepOrphanedFiles     bool
	Lazy                  bool
}

func Run(ctx context.Context, log *slog.Logger, args Arguments) (err error) {
	return NewGenerate(log, args).Run(ctx)
}
