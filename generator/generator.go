package generator

import (
	"bytes"
	"io"
	"strings"
	"time"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
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

func WithExtractStrings() GenerateOpt {
	return func(g *generator) error {
		g.w.literalWriter = &watchLiteralWriter{
			builder: &strings.Builder{},
		}
		return nil
	}
}

// WithSkipCodeGeneratedComment skips the code generated comment at the top of the file.
// gopls disables edit related functionality for generated files, so the templ LSP may
// wish to skip generation of this comment so that gopls provides expected results.
func WithSkipCodeGeneratedComment() GenerateOpt {
	return func(g *generator) error {
		g.skipCodeGeneratedComment = true
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
	// style to use for the generated HTML.
	style string
	// the contents to be syntax highlighted.
	contents []byte
	// packageName to use in the generated code.
	packageName string
	// componentName to use in the generated code.
	componentName string
	// skipCodeGeneratedComment skips the code generated comment at the top of the file.
	skipCodeGeneratedComment bool
}

type Config struct {
	HTMLOpts      []html.Option
	Style         string
	Contents      []byte
	PackageName   string
	ComponentName string
}

func Generate(w io.Writer, config Config, opts ...GenerateOpt) (literals string, err error) {

	g := generator{
		f:             html.New(config.HTMLOpts...),
		w:             NewRangeWriter(w),
		style:         config.Style,
		contents:      config.Contents,
		packageName:   config.PackageName,
		componentName: config.ComponentName,
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
	if err = g.writeImports(); err != nil {
		return
	}
	if err = g.writeComponent(); err != nil {
		return
	}
	if err = g.writeBlankAssignmentForRuntimeImport(); err != nil {
		return
	}

	return err
}

// See https://pkg.go.dev/cmd/go#hdr-Generate_Go_files_by_processing_source
// Automatically generated files have a comment in the header that instructs the LSP
// to stop operating.
func (g *generator) writeCodeGeneratedComment() (err error) {
	if g.skipCodeGeneratedComment {
		// Write an empty comment so that the file is the same shape.
		_, err = g.w.Write("//\n\n")
		return err
	}
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

func (g *generator) writePackage() (err error) {
	if _, err := g.w.Write("package " + g.packageName + "\n\n"); err != nil {
		return err
	}
	if _, err = g.w.Write("//lint:file-ignore SA4006 This context is only used if a nested component is present.\n\n"); err != nil {
		return err
	}
	return err
}

func (g *generator) writeImports() error {
	var err error
	// Always import templ because it's the interface type of all templates.
	if _, err = g.w.Write("import \"github.com/a-h/templ\"\n"); err != nil {
		return err
	}
	if _, err = g.w.Write("import templruntime \"github.com/a-h/templ/runtime\"\n"); err != nil {
		return err
	}
	if _, err = g.w.Write("\n"); err != nil {
		return err
	}
	return nil
}

func (g *generator) writeComponent() (err error) {
	if _, err = g.w.Write("func " + g.componentName + "() templ.Component {\n"); err != nil {
		return
	}
	if _, err = g.w.Write("\treturn templruntime.GeneratedTemplate(func(templ_7745c5c3_Input templruntime.GeneratedComponentInput) (templ_7745c5c3_Err error) {\n"); err != nil {
		return
	}
	if _, err = g.w.Write("\t\ttempl_7745c5c3_W, ctx := templ_7745c5c3_Input.Writer, templ_7745c5c3_Input.Context\n"); err != nil {
		return
	}
	if _, err = g.w.Write("\t\tif templ_7745c5c3_CtxErr := ctx.Err(); templ_7745c5c3_CtxErr != nil {\n"); err != nil {
		return
	}
	if _, err = g.w.Write("\t\t\treturn templ_7745c5c3_CtxErr\n"); err != nil {
		return
	}
	if _, err = g.w.Write("\t\t}\n"); err != nil {
		return
	}
	if _, err = g.w.Write("\t\ttempl_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templruntime.GetBuffer(templ_7745c5c3_W)\n"); err != nil {
		return
	}
	if _, err = g.w.Write("\t\tif !templ_7745c5c3_IsBuffer {\n"); err != nil {
		return
	}
	if _, err = g.w.Write("\t\t\tdefer func() {\n"); err != nil {
		return
	}
	if _, err = g.w.Write("\t\t\t\ttempl_7745c5c3_BufErr := templruntime.ReleaseBuffer(templ_7745c5c3_Buffer)\n"); err != nil {
		return
	}
	if _, err = g.w.Write("\t\t\t\tif templ_7745c5c3_Err == nil {\n"); err != nil {
		return
	}
	if _, err = g.w.Write("\t\t\t\t\ttempl_7745c5c3_Err = templ_7745c5c3_BufErr\n"); err != nil {
		return
	}
	if _, err = g.w.Write("\t\t\t\t}\n"); err != nil {
		return
	}
	if _, err = g.w.Write("\t\t\t}()\n"); err != nil {
		return
	}
	if _, err = g.w.Write("\t\t}\n"); err != nil {
		return
	}
	if _, err = g.w.Write("\t\tctx = templ.InitializeContext(ctx)\n"); err != nil {
		return
	}
	if _, err = g.w.Write("\t\ttempl_7745c5c3_Var1 := templ.GetChildren(ctx)\n"); err != nil {
		return
	}
	if _, err = g.w.Write("\t\tif templ_7745c5c3_Var1 == nil {\n"); err != nil {
		return
	}
	if _, err = g.w.Write("\t\t\ttempl_7745c5c3_Var1 = templ.NopComponent\n"); err != nil {
		return
	}
	if _, err = g.w.Write("\t\t}\n"); err != nil {
		return
	}
	if _, err = g.w.Write("\t\tctx = templ.ClearChildren(ctx)\n"); err != nil {
		return
	}

	chromaString, err := g.chroma()
	if err != nil {
		return err
	}

	if _, err = g.w.Write("\t\t_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(\"" + chromaString + "\")\n"); err != nil {
		return
	}
	if _, err = g.w.Write("\t\tif templ_7745c5c3_Err != nil {\n"); err != nil {
		return
	}
	if _, err = g.w.Write("\t\t\treturn templ_7745c5c3_Err\n"); err != nil {
		return
	}
	if _, err = g.w.Write("\t\t}\n"); err != nil {
		return
	}
	if _, err = g.w.Write("\t\treturn templ_7745c5c3_Err\n"); err != nil {
		return
	}
	if _, err = g.w.Write("\t})\n"); err != nil {
		return
	}
	if _, err = g.w.Write("}\n"); err != nil {
		return
	}
	return nil
}

func (g *generator) chroma() (s string, err error) {
	contents, err := io.ReadAll(bytes.NewReader(g.contents))
	if err != nil {
		return s, err
	}

	strContents := string(contents)

	lexer := lexers.Analyse(strContents)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Get(g.style)
	if style == nil {
		style = styles.Fallback
	}

	iterator, err := lexer.Tokenise(nil, strContents)

	var b bytes.Buffer
	ew := NewEscapeWriter(&b)
	if err := g.f.Format(ew, style, iterator); err != nil {
		return s, err
	}

	return b.String(), nil
}

// writeBlankAssignmentForRuntimeImport writes out a blank identifier assignment.
// This ensures that even if the github.com/a-h/templ/runtime package is not used in the generated code,
// the Go compiler will not complain about the unused import.
func (g *generator) writeBlankAssignmentForRuntimeImport() error {
	var err error
	if _, err = g.w.Write("var _ = templruntime.GeneratedTemplate"); err != nil {
		return err
	}
	return nil
}
