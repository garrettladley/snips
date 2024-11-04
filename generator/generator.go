package generator

import (
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters/html"
)

type GenerateOpt func(g *generator) error

// WithVersion enables the version to be included in the generated code.
func WithVersion(v string) GenerateOpt {
	return func(g *generator) error {
		g.version = v
		return nil
	}
}

// WithTimestamp enables the generated date to be included in the generated code.
func WithTimestamp(d time.Time) GenerateOpt {
	return func(g *generator) error {
		g.generatedDate = d.Format(time.RFC3339)
		return nil
	}
}

// WithFileName sets the filename of the templ file in template rendering error messages.
func WithFileName(name string) GenerateOpt {
	return func(g *generator) error {
		if filepath.IsAbs(name) {
			_, g.fileName = filepath.Split(name)
			return nil
		}
		g.fileName = name
		return nil
	}
}

func WithExtractStrings() GenerateOpt {
	return func(g *generator) error {
		g.w.literalWriter = &watchLiteralWriter{
			builder: &strings.Builder{},
		}
		return nil
	}
}

type generator struct {
	f chroma.Formatter
	w *RangeWriter

	// version of templ.
	version string
	// generatedDate to include as a comment.
	generatedDate string
	// fileName to include as a comment.
	fileName string
}

func Generate(w io.Writer, htmlOpts []html.Option, opts ...GenerateOpt) (literals string, err error) {
	g := generator{
		f: html.New(htmlOpts...),
		w: NewRangeWriter(w),
	}

	for _, opt := range opts {
		if err = opt(&g); err != nil {
			return
		}
	}

	err = g.generate()
	literals = g.w.literalWriter.literals()
	return
}

func (g *generator) generate() (err error) {
	if err = g.writeCodeGeneratedComment(); err != nil {
		return
	}
	if err = g.writeVersionComment(); err != nil {
		return
	}
	if err = g.writeGeneratedDateComment(); err != nil {
		return
	}
	if err = g.writePackage(); err != nil {
		return
	}

	return err
}

// See https://pkg.go.dev/cmd/go#hdr-Generate_Go_files_by_processing_source
// Automatically generated files have a comment in the header that instructs the LSP
// to stop operating.
func (g *generator) writeCodeGeneratedComment() (err error) {
	_, err = g.w.Write("// Code generated by snips - DO NOT EDIT.\n\n")
	return err
}

func (g *generator) writeVersionComment() (err error) {
	if g.version != "" {
		_, err = g.w.Write("// snips: version: " + g.version + "\n")
	}
	return err
}

func (g *generator) writeGeneratedDateComment() (err error) {
	if g.generatedDate != "" {
		_, err = g.w.Write("// snips: generated: " + g.generatedDate + "\n")
	}
	return err
}

func (g *generator) writePackage() error {
	// package ...
	_, err := g.w.Write("package snips")
	return err
}
