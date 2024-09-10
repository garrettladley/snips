package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"

	"github.com/fatih/color"
	"github.com/garrettladley/snips"
	"github.com/garrettladley/snips/cmd/snips/generatecmd"
	"github.com/garrettladley/snips/cmd/snips/sloghandler"
)

func main() {
	code := run(os.Stdout, os.Stderr, os.Args)
	if code != 0 {
		os.Exit(code)
	}
}

const usageText = `usage: snips <command> [<args>...]

snips - generate syntax highlighted templ components from code snippets


commands:
  generate   Generates syntax highlighted templ files from source code
  version    Prints the version
`

func run(stdout, stderr io.Writer, args []string) (code int) {
	if len(args) < 2 {
		fmt.Fprint(stderr, usageText)
		return 64 // EX_USAGE
	}
	switch args[1] {
	case "generate":
		return generateCmd(stdout, stderr, args[2:])
	case "version", "--version":
		fmt.Fprintln(stdout, snips.Version())
		return 0
	case "help", "-help", "--help", "-h":
		fmt.Fprint(stdout, usageText)
		return 0
	}
	fmt.Fprint(stderr, usageText)
	return 64 // EX_USAGE
}

func newLogger(logLevel string, verbose bool, stderr io.Writer) *slog.Logger {
	if verbose {
		logLevel = "debug"
	}
	level := slog.LevelInfo.Level()
	switch logLevel {
	case "debug":
		level = slog.LevelDebug.Level()
	case "warn":
		level = slog.LevelWarn.Level()
	case "error":
		level = slog.LevelError.Level()
	}
	return slog.New(sloghandler.NewHandler(stderr, &slog.HandlerOptions{
		AddSource: logLevel == "debug",
		Level:     level,
	}))
}

const generateUsageText = `usage: snips generate [<args>...]

Generates syntax highlighted templ components from code snippets.

Args:
  -stdout
    Prints to stdout instead of writing generated files to the filesystem.
    Only applicable when -f is used.
  -v
    Set log verbosity level to "debug". (default "info")
  -log-level
    Set log verbosity level. (default "info", options: "debug", "info", "warn", "error")
  -help
    Print help and exit.

Examples:

  // TODO
`

func generateCmd(stdout, stderr io.Writer, args []string) (code int) {
	cmd := flag.NewFlagSet("generate", flag.ExitOnError)
	toStdoutFlag := cmd.Bool("stdout", false, "")
	verboseFlag := cmd.Bool("v", false, "")
	logLevelFlag := cmd.String("log-level", "info", "")
	helpFlag := cmd.Bool("help", false, "")
	err := cmd.Parse(args)
	if err != nil {
		fmt.Fprint(stderr, generateUsageText)
		return 64 // EX_USAGE
	}
	if *helpFlag {
		fmt.Fprint(stdout, generateUsageText)
		return
	}

	log := newLogger(*logLevelFlag, *verboseFlag, stderr)

	ctx, cancel := context.WithCancel(context.Background())
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		<-signalChan
		fmt.Fprintln(stderr, "Stopping...")
		cancel()
	}()

	var fw generatecmd.FileWriterFunc
	if *toStdoutFlag {
		fw = generatecmd.WriterFileWriter(stdout)
	}

	err = generatecmd.Run(ctx, log, generatecmd.Arguments{
		FileWriter: fw,
	})
	if err != nil {
		color.New(color.FgRed).Fprint(stderr, "(✗) ")
		fmt.Fprintln(stderr, "Command failed: "+err.Error())
		return 1
	}
	return 0
}
