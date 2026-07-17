package compose

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// candidateNames lists the standard Docker Compose file names in priority order.
var candidateNames = []string{
	"docker-compose.yml",
	"docker-compose.yaml",
	"compose.yml",
	"compose.yaml",
}

// Project describes a detected Docker Compose project.
type Project struct {
	File string // absolute path to the compose file dmon is scoped to
	Dir  string // absolute path to the directory containing File
	Name string // resolved project name (for display and grouping)
}

// DetectComposeFile returns the absolute path to the compose file to use.
//
// If explicit is non-empty it is stat-checked and returned as an absolute path.
// Otherwise the function searches dir for files named by candidateNames in order
// and returns the first one found.
// Returns an error if the explicit file does not exist or if no candidate is found.
func DetectComposeFile(explicit, dir string) (string, error) {
	if explicit != "" {
		return resolveExplicit(explicit)
	}
	if abs, ok := scanDir(dir); ok {
		return abs, nil
	}
	return "", fmt.Errorf("no compose file found in %q (tried: %s)", dir, strings.Join(candidateNames, ", "))
}

// Detect resolves the compose project dmon should scope to, if any.
//
// Unlike DetectComposeFile, a failed auto-detection is not an error: it returns
// (nil, nil) so dmon can run from any directory and simply show all containers.
// An explicit file that does not exist is still a hard error.
func Detect(explicit, dir string) (*Project, error) {
	if explicit != "" {
		abs, err := resolveExplicit(explicit)
		if err != nil {
			return nil, err
		}
		return newProject(abs), nil
	}
	if abs, ok := scanDir(dir); ok {
		return newProject(abs), nil
	}
	return nil, nil
}

// resolveExplicit turns an explicit compose-file path into a verified absolute path.
func resolveExplicit(explicit string) (string, error) {
	abs, err := filepath.Abs(explicit)
	if err != nil {
		return "", fmt.Errorf("resolving compose file path %q: %w", explicit, err)
	}
	if _, err := os.Stat(abs); err != nil {
		return "", fmt.Errorf("compose file %q not found: %w", abs, err)
	}
	return abs, nil
}

// scanDir looks for a candidate compose file in dir, returning its absolute path.
func scanDir(dir string) (string, bool) {
	for _, name := range candidateNames {
		candidate := filepath.Join(dir, name)
		if _, err := os.Stat(candidate); err == nil {
			if abs, err := filepath.Abs(candidate); err == nil {
				return abs, true
			}
		}
	}
	return "", false
}

// newProject builds a Project from an absolute compose-file path.
func newProject(file string) *Project {
	dir := filepath.Dir(file)
	return &Project{
		File: file,
		Dir:  dir,
		Name: resolveName(file, dir),
	}
}

// nameLine matches a top-level `name:` key in a compose file.
var nameLine = regexp.MustCompile(`^name:\s*["']?([A-Za-z0-9][A-Za-z0-9_.-]*)["']?\s*$`)

// invalidNameChars matches characters not allowed in a normalized project name.
var invalidNameChars = regexp.MustCompile(`[^a-z0-9_-]+`)

// resolveName determines the project name the way Docker Compose does, in order:
// the COMPOSE_PROJECT_NAME env var, a top-level `name:` in the file, then a
// normalized form of the directory basename.
func resolveName(file, dir string) string {
	if env := strings.TrimSpace(os.Getenv("COMPOSE_PROJECT_NAME")); env != "" {
		return env
	}
	if name := nameFromFile(file); name != "" {
		return name
	}
	return normalizeName(filepath.Base(dir))
}

// nameFromFile returns the value of a top-level `name:` key, or "" if absent.
func nameFromFile(file string) string {
	f, err := os.Open(file)
	if err != nil {
		return ""
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if m := nameLine.FindStringSubmatch(sc.Text()); m != nil {
			return m[1]
		}
	}
	return ""
}

// normalizeName lower-cases s and collapses invalid characters to match the
// project name Compose derives from a directory.
func normalizeName(s string) string {
	s = invalidNameChars.ReplaceAllString(strings.ToLower(s), "")
	return strings.Trim(s, "_-")
}
