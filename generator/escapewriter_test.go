package generator

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestEscapeWriter(t *testing.T) {
	t.Run("writes unescaped characters unchanged", func(t *testing.T) {
		w := new(bytes.Buffer)
		ew := NewEscapeWriter(w)

		input := []byte("hello world")
		expected := "hello world"

		n, err := ew.Write(input)
		if err != nil {
			t.Fatalf("failed to write: %v", err)
		}
		if n != len(input) {
			t.Errorf("expected to write %d bytes, wrote %d", len(input), n)
		}
		if diff := cmp.Diff(expected, w.String()); diff != "" {
			t.Errorf("unexpected output (-want +got):\n%s", diff)
		}
	})

	t.Run("escapes double quotes", func(t *testing.T) {
		w := new(bytes.Buffer)
		ew := NewEscapeWriter(w)

		input := []byte(`"quoted text"`)
		expected := `\"quoted text\"`

		n, err := ew.Write(input)
		if err != nil {
			t.Fatalf("failed to write: %v", err)
		}
		if n != len(expected) {
			t.Errorf("expected to write %d bytes, wrote %d", len(input), n)
		}
		if diff := cmp.Diff(expected, w.String()); diff != "" {
			t.Errorf("unexpected output (-want +got):\n%s", diff)
		}
	})

	t.Run("escapes newlines", func(t *testing.T) {
		w := new(bytes.Buffer)
		ew := NewEscapeWriter(w)

		input := []byte("line1\nline2\nline3")
		expected := `line1\nline2\nline3`

		n, err := ew.Write(input)
		if err != nil {
			t.Fatalf("failed to write: %v", err)
		}
		if n != len(expected) {
			t.Errorf("expected to write %d bytes, wrote %d", len(input), n)
		}
		if diff := cmp.Diff(expected, w.String()); diff != "" {
			t.Errorf("unexpected output (-want +got):\n%s", diff)
		}
	})

	t.Run("handles mixed escape sequences", func(t *testing.T) {
		w := new(bytes.Buffer)
		ew := NewEscapeWriter(w)

		input := []byte("\"Hello\nWorld\"")
		expected := `\"Hello\nWorld\"`

		n, err := ew.Write(input)
		if err != nil {
			t.Fatalf("failed to write: %v", err)
		}
		if n != len(expected) {
			t.Errorf("expected to write %d bytes, wrote %d", len(input), n)
		}
		if diff := cmp.Diff(expected, w.String()); diff != "" {
			t.Errorf("unexpected output (-want +got):\n%s", diff)
		}
	})

	t.Run("handles empty input", func(t *testing.T) {
		w := new(bytes.Buffer)
		ew := NewEscapeWriter(w)

		input := []byte("")
		expected := ""

		n, err := ew.Write(input)
		if err != nil {
			t.Fatalf("failed to write: %v", err)
		}
		if n != 0 {
			t.Errorf("expected to write 0 bytes, wrote %d", n)
		}
		if diff := cmp.Diff(expected, w.String()); diff != "" {
			t.Errorf("unexpected output (-want +got):\n%s", diff)
		}
	})
}
