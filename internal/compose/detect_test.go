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

func TestDetect(t *testing.T) {
	t.Run("auto-detect returns a project", func(t *testing.T) {
		dir := t.TempDir()
		f := filepath.Join(dir, "docker-compose.yml")
		if err := os.WriteFile(f, []byte("services: {}"), 0o644); err != nil {
			t.Fatal(err)
		}
		p, err := Detect("", dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p == nil {
			t.Fatal("expected a project, got nil")
		}
		abs, _ := filepath.Abs(f)
		if p.File != abs {
			t.Errorf("File = %q, want %q", p.File, abs)
		}
		if p.Dir != filepath.Dir(abs) {
			t.Errorf("Dir = %q, want %q", p.Dir, filepath.Dir(abs))
		}
	})

	t.Run("no compose file is not an error (run from anywhere)", func(t *testing.T) {
		dir := t.TempDir()
		p, err := Detect("", dir)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if p != nil {
			t.Errorf("expected nil project, got %+v", p)
		}
	})

	t.Run("explicit missing file is still an error", func(t *testing.T) {
		dir := t.TempDir()
		_, err := Detect(filepath.Join(dir, "nope.yml"), dir)
		if err == nil {
			t.Error("expected error for missing explicit file")
		}
	})

	t.Run("name from top-level name key", func(t *testing.T) {
		dir := t.TempDir()
		f := filepath.Join(dir, "docker-compose.yml")
		if err := os.WriteFile(f, []byte("name: my-stack\nservices: {}"), 0o644); err != nil {
			t.Fatal(err)
		}
		p, err := Detect("", dir)
		if err != nil {
			t.Fatal(err)
		}
		if p.Name != "my-stack" {
			t.Errorf("Name = %q, want %q", p.Name, "my-stack")
		}
	})

	t.Run("name falls back to normalized dir basename", func(t *testing.T) {
		t.Setenv("COMPOSE_PROJECT_NAME", "")
		parent := t.TempDir()
		dir := filepath.Join(parent, "My App")
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		f := filepath.Join(dir, "compose.yml")
		if err := os.WriteFile(f, []byte("services: {}"), 0o644); err != nil {
			t.Fatal(err)
		}
		p, err := Detect("", dir)
		if err != nil {
			t.Fatal(err)
		}
		if p.Name != "myapp" {
			t.Errorf("Name = %q, want %q", p.Name, "myapp")
		}
	})

	t.Run("COMPOSE_PROJECT_NAME wins", func(t *testing.T) {
		t.Setenv("COMPOSE_PROJECT_NAME", "override")
		dir := t.TempDir()
		f := filepath.Join(dir, "docker-compose.yml")
		if err := os.WriteFile(f, []byte("name: from-file\nservices: {}"), 0o644); err != nil {
			t.Fatal(err)
		}
		p, err := Detect("", dir)
		if err != nil {
			t.Fatal(err)
		}
		if p.Name != "override" {
			t.Errorf("Name = %q, want %q", p.Name, "override")
		}
	})
}
