package generatecmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"go/format"
	"io"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/fsnotify/fsnotify"
	"github.com/garrettladley/snips"
	"github.com/garrettladley/snips/generator"
)

type FileWriterFunc func(name string, contents []byte) error

func FileWriter(fileName string, contents []byte) error {
	return os.WriteFile(fileName, contents, 0o644)
}

func WriterFileWriter(w io.Writer) FileWriterFunc {
	return func(_ string, contents []byte) error {
		_, err := w.Write(contents)
		return err
	}
}

func NewFSEventHandler(
	log *slog.Logger,
	dir string,
	devMode bool,
	genOpts []html.Option,
	keepOrphanedFiles bool,
	fileWriter FileWriterFunc,
	lazy bool,
) *FSEventHandler {
	if !path.IsAbs(dir) {
		dir, _ = filepath.Abs(dir)
	}
	fseh := &FSEventHandler{
		Log:                        log,
		dir:                        dir,
		fileNameToLastModTime:      make(map[string]time.Time),
		fileNameToLastModTimeMutex: &sync.Mutex{},
		fileNameToError:            make(map[string]struct{}),
		fileNameToErrorMutex:       &sync.Mutex{},
		hashes:                     make(map[string][sha256.Size]byte),
		hashesMutex:                &sync.Mutex{},
		genOpts:                    genOpts,
		DevMode:                    devMode,
		keepOrphanedFiles:          keepOrphanedFiles,
		writer:                     fileWriter,
		lazy:                       lazy,
	}
	if devMode {
		// fseh.genOpts = append(fseh.genOpts, generator.WithExtractStrings())
	}
	return fseh
}

type FSEventHandler struct {
	Log *slog.Logger
	// dir is the root directory being processed.
	dir                        string
	fileNameToLastModTime      map[string]time.Time
	fileNameToLastModTimeMutex *sync.Mutex
	fileNameToError            map[string]struct{}
	fileNameToErrorMutex       *sync.Mutex
	hashes                     map[string][sha256.Size]byte
	hashesMutex                *sync.Mutex
	genOpts                    []html.Option
	genSourceMapVis            bool
	DevMode                    bool
	Errors                     []error
	keepOrphanedFiles          bool
	writer                     func(string, []byte) error
	lazy                       bool
}

func (h *FSEventHandler) HandleEvent(ctx context.Context, event fsnotify.Event) (goUpdated, textUpdated bool, err error) {
	// Handle _code.txt files.
	if !event.Has(fsnotify.Remove) && strings.HasSuffix(event.Name, "_code.txt") {
		if h.DevMode {
			// Don't delete the file if we're in dev mode, but mark that text was updated.
			return false, true, nil
		}
		h.Log.Debug("Deleting watch mode file", slog.String("file", event.Name))
		if err = os.Remove(event.Name); err != nil {
			h.Log.Warn("Failed to remove watch mode text file", slog.Any("error", err))
			return false, false, nil
		}
		return false, false, nil
	}

	// Handle .code.* files.
	if !snips.ContainsDotCodeDot(event.Name) {
		return false, false, nil
	}

	// If the file hasn't been updated since the last time we processed it, ignore it.
	_, updatedModTime := h.UpsertLastModTime(event.Name)
	if !updatedModTime {
		h.Log.Debug("Skipping file because it wasn't updated", slog.String("file", event.Name))
		return false, false, nil
	}

	// Start a processor.
	start := time.Now()
	goUpdated, textUpdated, err = h.generate(event.Name)
	if err != nil {
		h.Log.Error(
			"Error generating code",
			slog.String("file", event.Name),
			slog.Any("error", err),
		)
		h.SetError(event.Name, true)
		return goUpdated, textUpdated, fmt.Errorf("failed to generate code for %q: %w", event.Name, err)
	}

	if errorCleared, errorCount := h.SetError(event.Name, false); errorCleared {
		h.Log.Info("Error cleared", slog.String("file", event.Name), slog.Int("errors", errorCount))
	}
	h.Log.Debug("Generated code", slog.String("file", event.Name), slog.Duration("in", time.Since(start)))

	return goUpdated, textUpdated, nil
}

func (h *FSEventHandler) SetError(fileName string, hasError bool) (previouslyHadError bool, errorCount int) {
	h.fileNameToErrorMutex.Lock()
	defer h.fileNameToErrorMutex.Unlock()
	_, previouslyHadError = h.fileNameToError[fileName]
	delete(h.fileNameToError, fileName)
	if hasError {
		h.fileNameToError[fileName] = struct{}{}
	}
	return previouslyHadError, len(h.fileNameToError)
}

func (h *FSEventHandler) UpsertLastModTime(fileName string) (modTime time.Time, updated bool) {
	fileInfo, err := os.Stat(fileName)
	if err != nil {
		return modTime, false
	}
	h.fileNameToLastModTimeMutex.Lock()
	defer h.fileNameToLastModTimeMutex.Unlock()
	previousModTime := h.fileNameToLastModTime[fileName]
	currentModTime := fileInfo.ModTime()
	if !currentModTime.After(previousModTime) {
		return currentModTime, false
	}
	h.fileNameToLastModTime[fileName] = currentModTime
	return currentModTime, true
}

func (h *FSEventHandler) UpsertHash(fileName string, hash [sha256.Size]byte) (updated bool) {
	h.hashesMutex.Lock()
	defer h.hashesMutex.Unlock()
	lastHash := h.hashes[fileName]
	if lastHash == hash {
		return false
	}
	h.hashes[fileName] = hash
	return true
}

// generate Go code for a single template.
// If a basePath is provided, the filename included in error messages is relative to it.
func (h *FSEventHandler) generate(fileName string) (goUpdated, textUpdated bool, err error) {
	pc, err := from(fileName)
	if err != nil {
		return false, false, fmt.Errorf("failed to parse path %q: %w", fileName, err)
	}

	f, err := os.ReadFile(fileName)
	if err != nil {
		return false, false, fmt.Errorf("failed to open %q: %w", fileName, err)
	}

	var b bytes.Buffer
	literals, err := generator.Generate(&b,
		generator.Config{
			HTMLOpts:      h.genOpts,
			Style:         "", // TODO: drill down
			Contents:      f,
			PackageName:   pc.packageName,
			ComponentName: pc.componentName,
		})
	if err != nil {
		return false, false, fmt.Errorf("%s generation error: %w", fileName, err)
	}

	formattedGoCode, err := format.Source(b.Bytes())
	if err != nil {
		return false, false, fmt.Errorf("% source formatting error %w", fileName, err)
	}

	targetFileName := fileName + "_templ.go"
	// Hash output, and write out the file if the codeHash has changed.
	codeHash := sha256.Sum256(formattedGoCode)
	if h.UpsertHash(targetFileName, codeHash) {
		goUpdated = true
		if err = h.writer(targetFileName, formattedGoCode); err != nil {
			return false, false, fmt.Errorf("failed to write target file %q: %w", targetFileName, err)
		}
	}

	// Add the txt file if it has changed.
	if len(literals) > 0 {
		txtFileName := "_code.txt"
		txtHash := sha256.Sum256([]byte(literals))
		if h.UpsertHash(txtFileName, txtHash) {
			textUpdated = true
			if err = os.WriteFile(txtFileName, []byte(literals), 0o644); err != nil {
				return false, false, fmt.Errorf("failed to write string literal file %q: %w", txtFileName, err)
			}
		}
	}

	return goUpdated, textUpdated, err
}

type packageComponent struct {
	packageName   string
	componentName string
}

func from(fileName string) (pc packageComponent, err error) {
	fileName = stripCode(fileName)
	parts := strings.Split(filepath.ToSlash(fileName), "/")
	if len(parts) == 0 {
		return pc, fmt.Errorf("unexpected file name %q", fileName)
	}

	pc.componentName = sanitze(parts[len(parts)-1])
	pc.packageName = snips.PackageName(strings.Join(parts[:len(parts)-1], "/"))
	return
}

func stripCode(fileName string) string {
	parts := strings.Split(fileName, ".code")
	if len(parts) != 2 {
		return fileName
	}
	return parts[0] + parts[1]
}

func sanitze(fileName string) string {
	var result []rune
	firstLetter := true
	for _, char := range fileName {
		if char == ' ' {
			firstLetter = true
			continue
		}

		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			if firstLetter {
				result = append(result, unicode.ToUpper(char))
				firstLetter = false
			} else {
				result = append(result, char)
			}
		} else {
			firstLetter = true
		}
	}
	return string(result)
}
