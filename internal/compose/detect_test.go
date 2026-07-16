package compose

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectComposeFile(t *testing.T) {
	t.Run("explicit file exists", func(t *testing.T) {
		dir := t.TempDir()
		f := filepath.Join(dir, "my-compose.yml")
		if err := os.WriteFile(f, []byte("version: '3'"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := DetectComposeFile(f, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		abs, _ := filepath.Abs(f)
		if got != abs {
			t.Errorf("got %q, want %q", got, abs)
		}
	})

	t.Run("explicit file missing", func(t *testing.T) {
		dir := t.TempDir()
		_, err := DetectComposeFile(filepath.Join(dir, "nonexistent.yml"), dir)
		if err == nil {
			t.Error("expected error for missing explicit file")
		}
	})

	t.Run("auto-detect docker-compose.yml", func(t *testing.T) {
		dir := t.TempDir()
		f := filepath.Join(dir, "docker-compose.yml")
		if err := os.WriteFile(f, []byte("version: '3'"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := DetectComposeFile("", dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		abs, _ := filepath.Abs(f)
		if got != abs {
			t.Errorf("got %q, want %q", got, abs)
		}
	})

	t.Run("auto-detect compose.yaml preferred over later candidates", func(t *testing.T) {
		dir := t.TempDir()
		// Only compose.yaml exists (the third candidate)
		f := filepath.Join(dir, "compose.yaml")
		if err := os.WriteFile(f, []byte("version: '3'"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := DetectComposeFile("", dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		abs, _ := filepath.Abs(f)
		if got != abs {
			t.Errorf("got %q, want %q", got, abs)
		}
	})

	t.Run("priority: docker-compose.yml beats compose.yml", func(t *testing.T) {
		dir := t.TempDir()
		first := filepath.Join(dir, "docker-compose.yml")
		second := filepath.Join(dir, "compose.yml")
		for _, f := range []string{first, second} {
			if err := os.WriteFile(f, []byte("version: '3'"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		got, err := DetectComposeFile("", dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		abs, _ := filepath.Abs(first)
		if got != abs {
			t.Errorf("got %q, want %q (first candidate should win)", got, abs)
		}
	})

	t.Run("no compose file found", func(t *testing.T) {
		dir := t.TempDir()
		_, err := DetectComposeFile("", dir)
		if err == nil {
			t.Error("expected error when no compose file present")
		}
	})
}
