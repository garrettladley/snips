package generator

import "io"

type EscapeWriter struct {
	w io.Writer
}

func NewEscapeWriter(w io.Writer) *EscapeWriter {
	return &EscapeWriter{w: w}
}

func (w *EscapeWriter) Write(p []byte) (n int, err error) {
	var processed []byte
	for i := 0; i < len(p); i++ {
		switch p[i] {
		case '"':
			processed = append(processed, '\\', '"')
		case '\n':
			processed = append(processed, '\\', 'n')
		default:
			processed = append(processed, p[i])
		}
	}

	return w.w.Write(processed)
}
