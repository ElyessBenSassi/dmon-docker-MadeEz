package compose

import (
	"fmt"
	"os"
	"path/filepath"
)

// candidateNames lists the standard Docker Compose file names in priority order.
var candidateNames = []string{
	"docker-compose.yml",
	"docker-compose.yaml",
	"compose.yml",
	"compose.yaml",
}

// DetectComposeFile returns the absolute path to the compose file to use.
//
// If explicit is non-empty it is stat-checked and returned as an absolute path.
// Otherwise the function searches dir for files named by candidateNames in order
// and returns the first one found.
// Returns an error if the explicit file does not exist or if no candidate is found.
func DetectComposeFile(explicit, dir string) (string, error) {
	if explicit != "" {
		abs, err := filepath.Abs(explicit)
		if err != nil {
			return "", fmt.Errorf("resolving compose file path %q: %w", explicit, err)
		}
		if _, err := os.Stat(abs); err != nil {
			return "", fmt.Errorf("compose file %q not found: %w", abs, err)
		}
		return abs, nil
	}

	for _, name := range candidateNames {
		candidate := filepath.Join(dir, name)
		if _, err := os.Stat(candidate); err == nil {
			abs, err := filepath.Abs(candidate)
			if err != nil {
				return "", fmt.Errorf("resolving candidate path %q: %w", candidate, err)
			}
			return abs, nil
		}
	}

	return "", fmt.Errorf("no compose file found in %q (tried: docker-compose.yml, docker-compose.yaml, compose.yml, compose.yaml)", dir)
}
