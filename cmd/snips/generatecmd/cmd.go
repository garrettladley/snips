package generatecmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/fsnotify/fsnotify"
	"github.com/garrettladley/snips/cmd/snips/generatecmd/modcheck"
	"github.com/garrettladley/snips/cmd/snips/generatecmd/watcher"
)

func NewGenerate(log *slog.Logger, args Arguments) (g *Generate) {
	g = &Generate{
		Log:  log,
		Args: &args,
	}
	if g.Args.WorkerCount == 0 {
		g.Args.WorkerCount = runtime.NumCPU()
	}
	return g
}

type Generate struct {
	Log  *slog.Logger
	Args *Arguments
}

type GenerationEvent struct {
	Event       fsnotify.Event
	GoUpdated   bool
	TextUpdated bool
}

func (cmd Generate) Run(ctx context.Context) (err error) {
	if cmd.Args.Watch && cmd.Args.FileName != "" {
		return fmt.Errorf("cannot watch a single file, remove the -f or -watch flag")
	}
	writingToWriter := cmd.Args.FileWriter != nil
	if cmd.Args.FileName == "" && writingToWriter {
		return fmt.Errorf("only a single file can be output to stdout, add the -f flag to specify the file to generate code for")
	}
	// Default to writing to files.
	if cmd.Args.FileWriter == nil {
		cmd.Args.FileWriter = FileWriter
	}

	// Use absolute path.
	if !path.IsAbs(cmd.Args.Path) {
		cmd.Args.Path, err = filepath.Abs(cmd.Args.Path)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}
	}

	opts := []html.Option{
		html.TabWidth(cmd.Args.TabWidth),
		html.BaseLineNumber(cmd.Args.BaseLine),
		html.WithLineNumbers(cmd.Args.Lines),
		html.LineNumbersInTable(cmd.Args.LinesTable),
		html.WithLinkableLineNumbers(cmd.Args.LinkableLines, "L"),
	}

	// Check the version of the templ module.
	if err := modcheck.Check(cmd.Args.Path); err != nil {
		cmd.Log.Warn("templ version check: " + err.Error())
	}

	fseh := NewFSEventHandler(
		cmd.Log,
		cmd.Args.Path,
		cmd.Args.Watch,
		opts,
		cmd.Args.KeepOrphanedFiles,
		cmd.Args.FileWriter,
		cmd.Args.Lazy,
	)

	// If we're processing a single file, don't bother setting up the channels/multithreaing.
	if cmd.Args.FileName != "" {
		_, _, err = fseh.HandleEvent(ctx, fsnotify.Event{
			Name: cmd.Args.FileName,
			Op:   fsnotify.Create,
		})
		return err
	}

	// Start timer.
	start := time.Now()

	// Create channels:
	// For the initial filesystem walk and subsequent (optional) fsnotify events.
	events := make(chan fsnotify.Event)
	// Count of events currently being processed by the event handler.
	var eventsWG sync.WaitGroup
	// Used to check that the event handler has completed.
	var eventHandlerWG sync.WaitGroup
	// For errs from the watcher.
	errs := make(chan error)
	// Tracks whether errors occurred during the generation process.
	var errorCount atomic.Int64
	// For triggering actions after generation has completed.
	postGeneration := make(chan *GenerationEvent, 256)
	// Used to check that the post-generation handler has completed.
	var postGenerationWG sync.WaitGroup
	var postGenerationEventsWG sync.WaitGroup

	// Waitgroup for the push process.
	var pushHandlerWG sync.WaitGroup

	// Start process to push events into the channel.
	pushHandlerWG.Add(1)
	go func() {
		defer pushHandlerWG.Done()
		defer close(events)
		cmd.Log.Debug(
			"Walking directory",
			slog.String("path", cmd.Args.Path),
			slog.Bool("devMode", cmd.Args.Watch),
		)
		if err := watcher.WalkFiles(ctx, cmd.Args.Path, events); err != nil {
			cmd.Log.Error("WalkFiles failed, exiting", slog.Any("error", err))
			errs <- FatalError{Err: fmt.Errorf("failed to walk files: %w", err)}
			return
		}
		if !cmd.Args.Watch {
			cmd.Log.Debug("Dev mode not enabled, process can finish early")
			return
		}
		cmd.Log.Info("Watching files")
		rw, err := watcher.Recursive(ctx, cmd.Args.Path, events, errs)
		if err != nil {
			cmd.Log.Error("Recursive watcher setup failed, exiting", slog.Any("error", err))
			errs <- FatalError{Err: fmt.Errorf("failed to setup recursive watcher: %w", err)}
			return
		}
		cmd.Log.Debug("Waiting for context to be cancelled to stop watching files")
		<-ctx.Done()
		cmd.Log.Debug("Context cancelled, closing watcher")
		if err := rw.Close(); err != nil {
			cmd.Log.Error("Failed to close watcher", slog.Any("error", err))
		}
		cmd.Log.Debug("Waiting for events to be processed")
		eventsWG.Wait()
		cmd.Log.Debug(
			"All pending events processed, waiting for pending post-generation events to complete",
		)
		postGenerationEventsWG.Wait()
		cmd.Log.Debug(
			"All post-generation events processed, running walk again, but in production mode",
			slog.Int64("errorCount", errorCount.Load()),
		)
		// Reset to reprocess all files in production mode.
		fseh = NewFSEventHandler(
			cmd.Log,
			cmd.Args.Path,
			false, // Force production mode.
			opts,
			cmd.Args.KeepOrphanedFiles,
			cmd.Args.FileWriter,
			cmd.Args.Lazy,
		)
		errorCount.Store(0)
		if err := watcher.WalkFiles(ctx, cmd.Args.Path, events); err != nil {
			cmd.Log.Error("Post dev mode WalkFiles failed", slog.Any("error", err))
			errs <- FatalError{Err: fmt.Errorf("failed to walk files: %w", err)}
			return
		}
	}()

	// Start process to handle events.
	eventHandlerWG.Add(1)
	sem := make(chan struct{}, cmd.Args.WorkerCount)
	go func() {
		defer eventHandlerWG.Done()
		defer close(postGeneration)
		cmd.Log.Debug("Starting event handler")
		for event := range events {
			eventsWG.Add(1)
			sem <- struct{}{}
			go func(event fsnotify.Event) {
				cmd.Log.Debug("Processing file", slog.String("file", event.Name))
				defer eventsWG.Done()
				defer func() { <-sem }()
				goUpdated, textUpdated, err := fseh.HandleEvent(ctx, event)
				if err != nil {
					cmd.Log.Error("Event handler failed", slog.Any("error", err))
					errs <- err
				}
				if goUpdated || textUpdated {
					postGeneration <- &GenerationEvent{
						Event:       event,
						GoUpdated:   goUpdated,
						TextUpdated: textUpdated,
					}
				}
			}(event)
		}
		// Wait for all events to be processed before closing.
		eventsWG.Wait()
	}()

	// Start process to handle post-generation events.
	var updates int
	postGenerationWG.Add(1)
	go func() {
		defer close(errs)
		defer postGenerationWG.Done()
		cmd.Log.Debug("Starting post-generation handler")
		timeout := time.NewTimer(time.Hour * 24 * 365)
		var goUpdated, textUpdated bool
		for {
			select {
			case ge := <-postGeneration:
				if ge == nil {
					cmd.Log.Debug("Post-generation event channel closed, exiting")
					return
				}
				goUpdated = goUpdated || ge.GoUpdated
				textUpdated = textUpdated || ge.TextUpdated
				if goUpdated || textUpdated {
					updates++
				}
				// Reset timer.
				if !timeout.Stop() {
					<-timeout.C
				}
				timeout.Reset(time.Millisecond * 100)
			case <-timeout.C:
				if !goUpdated && !textUpdated {
					// Nothing to process, reset timer and wait again.
					timeout.Reset(time.Hour * 24 * 365)
					break
				}
				postGenerationEventsWG.Add(1)
				postGenerationEventsWG.Done()
				// Reset timer.
				timeout.Reset(time.Millisecond * 100)
				textUpdated = false
				goUpdated = false
			}
		}
	}()

	// Read errors.
	for err := range errs {
		if err == nil {
			continue
		}
		if errors.Is(err, FatalError{}) {
			cmd.Log.Debug("Fatal error, exiting")
			return err
		}
		cmd.Log.Error("Error", slog.Any("error", err))
		errorCount.Add(1)
	}

	// Wait for everything to complete.
	cmd.Log.Debug("Waiting for push handler to complete")
	pushHandlerWG.Wait()
	cmd.Log.Debug("Waiting for event handler to complete")
	eventHandlerWG.Wait()
	cmd.Log.Debug("Waiting for post-generation handler to complete")
	postGenerationWG.Wait()

	// Check for errors after everything has completed.
	if errorCount.Load() > 0 {
		return fmt.Errorf("generation completed with %d errors", errorCount.Load())
	}

	cmd.Log.Info(
		"Complete",
		slog.Int("updates", updates),
		slog.Duration("duration", time.Since(start)),
	)
	return nil
}
