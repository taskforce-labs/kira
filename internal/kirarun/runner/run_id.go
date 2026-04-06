package runner

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"kira/internal/kirarun/session"
)

// DeriveRunID returns scriptName-UTCtimestamp with second precision (e.g. myflow-20060102150405).
func DeriveRunID(scriptName string, t time.Time) (string, error) {
	base := strings.TrimSuffix(filepath.Base(scriptName), filepath.Ext(scriptName))
	base = strings.TrimSpace(base)
	if base == "" {
		return "", fmt.Errorf("empty workflow name")
	}
	for _, r := range base {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' && r != '.' {
			return "", fmt.Errorf("invalid character in workflow name %q", base)
		}
	}
	ts := t.UTC().Format("20060102150405")
	id := base + "-" + ts
	if err := session.ValidateRunID(id); err != nil {
		return "", err
	}
	return id, nil
}
