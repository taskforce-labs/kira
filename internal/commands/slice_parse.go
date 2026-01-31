// Package commands implements slice markdown parsing and generation.
package commands

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

const (
	slicesHeading       = "## Slices"
	requirementsHeading = "## Requirements"
	acceptanceHeading   = "## Acceptance Criteria"
)

var (
	taskLineRegex    = regexp.MustCompile(`^-\s+\[([ xX])\]\s+(T\d+):\s*(.*)$`)
	taskLineOpenDone = regexp.MustCompile(`^-\s+\[(open|done)\]\s+(T\d+):\s*(.*)$`)
	taskIDNumRegex   = regexp.MustCompile(`T(\d+)`)
	commitLineRegex  = regexp.MustCompile(`^Commit:\s*(.*)$`)
	notesLineRegex   = regexp.MustCompile(`^\s+-\s+Notes:\s*(.*)$`)
)

// ParseSlicesSection parses the ## Slices section from work item markdown.
// Returns slices with tasks; optional "Commit: ..." under ### Name; task lines
// - [ ] T001: desc (open) or - [x] T001: desc (done). Optional [open]/[done] format.
func ParseSlicesSection(content []byte) ([]Slice, error) {
	start, end, found := findSlicesSection(content)
	if !found {
		return nil, nil
	}
	section := content[start:end]
	lines := strings.Split(string(section), "\n")
	var slices []Slice
	var current *Slice
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "### ") {
			current = parseSliceHeading(lines, i, trimmed)
			slices = append(slices, *current)
			continue
		}
		if current == nil {
			continue
		}
		if task := parseTaskLine(lines, i, trimmed); task != nil {
			current.Tasks = append(current.Tasks, *task)
			slices[len(slices)-1] = *current
		}
	}
	return slices, nil
}

func parseSliceHeading(lines []string, i int, trimmed string) *Slice {
	name := strings.TrimSpace(trimmed[4:])
	s := &Slice{Name: name, Tasks: []Task{}}
	if i+1 < len(lines) {
		next := strings.TrimSpace(lines[i+1])
		if commitLineRegex.MatchString(next) {
			matches := commitLineRegex.FindStringSubmatch(next)
			if len(matches) >= 2 {
				s.CommitSummary = strings.TrimSpace(matches[1])
			}
		}
	}
	return s
}

func parseTaskLine(lines []string, i int, trimmed string) *Task {
	if taskLineRegex.MatchString(trimmed) {
		matches := taskLineRegex.FindStringSubmatch(trimmed)
		if len(matches) >= 4 {
			done := strings.ToLower(matches[1]) == "x"
			t := &Task{ID: matches[2], Description: strings.TrimSpace(matches[3]), Done: done}
			if i+1 < len(lines) && notesLineRegex.MatchString(lines[i+1]) {
				nm := notesLineRegex.FindStringSubmatch(lines[i+1])
				if len(nm) >= 2 {
					t.Notes = strings.TrimSpace(nm[1])
				}
			}
			return t
		}
	}
	if taskLineOpenDone.MatchString(trimmed) {
		matches := taskLineOpenDone.FindStringSubmatch(trimmed)
		if len(matches) >= 4 {
			done := strings.ToLower(matches[1]) == defaultReleaseStatus
			return &Task{ID: matches[2], Description: strings.TrimSpace(matches[3]), Done: done}
		}
	}
	return nil
}

// findSlicesSection returns start and end byte offsets of the ## Slices section (including heading and up to next ## or EOF).
func findSlicesSection(content []byte) (start, end int, found bool) {
	idx := bytes.Index(content, []byte(slicesHeading))
	if idx < 0 {
		return 0, 0, false
	}
	start = idx
	// Find end: next ## at start of line
	rest := content[start+len(slicesHeading):]
	nextH2 := bytes.Index(rest, []byte("\n## "))
	if nextH2 >= 0 {
		end = start + len(slicesHeading) + nextH2
	} else {
		end = len(content)
		// Trim trailing newlines from section so we don't duplicate
		for end > start && (content[end-1] == '\n' || content[end-1] == '\r') {
			end--
		}
	}
	return start, end, true
}

// GenerateSlicesSection formats slices as markdown (## Slices, ### Name, optional Commit:, task list).
func GenerateSlicesSection(slices []Slice, taskIDFormat string) []byte {
	_ = taskIDFormat // reserved for future use (e.g. custom ID format)
	var b strings.Builder
	b.WriteString(slicesHeading)
	b.WriteString("\n\n")
	for _, s := range slices {
		b.WriteString("### ")
		b.WriteString(s.Name)
		b.WriteString("\n")
		if s.CommitSummary != "" {
			b.WriteString("Commit: ")
			b.WriteString(s.CommitSummary)
			b.WriteString("\n")
		}
		for _, t := range s.Tasks {
			if t.Done {
				b.WriteString("- [x] ")
			} else {
				b.WriteString("- [ ] ")
			}
			b.WriteString(t.ID)
			b.WriteString(": ")
			b.WriteString(t.Description)
			b.WriteString("\n")
			if t.Notes != "" {
				b.WriteString("  - Notes: ")
				b.WriteString(t.Notes)
				b.WriteString("\n")
			}
		}
		b.WriteString("\n")
	}
	return []byte(strings.TrimRight(b.String(), "\n"))
}

// NextTaskID returns the next sequential task ID (e.g. T004) given existing slices and format (e.g. "T%03d").
func NextTaskID(slices []Slice, format string) (string, error) {
	if format == "" {
		format = "T%03d"
	}
	maxNum := 0
	for _, s := range slices {
		for _, t := range s.Tasks {
			m := taskIDNumRegex.FindStringSubmatch(t.ID)
			if len(m) >= 2 {
				var n int
				if _, err := fmt.Sscanf(m[1], "%d", &n); err == nil && n > maxNum {
					maxNum = n
				}
			}
		}
	}
	return fmt.Sprintf(format, maxNum+1), nil
}

// ReplaceSlicesSection replaces the existing ## Slices section with the new content, or inserts at best position.
// Insert after first ## Requirements or ## Acceptance Criteria, or at end of file.
func ReplaceSlicesSection(content, newSection []byte) ([]byte, error) {
	start, end, found := findSlicesSection(content)
	if found {
		before := content[:start]
		after := content[end:]
		// Preserve trailing newline after section if present
		if len(after) > 0 && (after[0] == '\n' || after[0] == '\r') {
			after = after[1:]
		}
		result := make([]byte, 0, len(before)+len(newSection)+len(after)+2)
		result = append(result, before...)
		result = append(result, newSection...)
		if len(after) > 0 {
			result = append(result, '\n')
			result = append(result, after...)
		}
		return result, nil
	}
	// Insert at best position: after ## Requirements or ## Acceptance Criteria
	insertAt := findInsertPosition(content)
	before := content[:insertAt]
	after := content[insertAt:]
	sep := "\n\n"
	if len(before) > 0 && !bytes.HasSuffix(before, []byte("\n")) {
		sep = "\n\n"
	}
	result := make([]byte, 0, len(before)+len(sep)+len(newSection)+len(after))
	result = append(result, before...)
	result = append(result, sep...)
	result = append(result, newSection...)
	if len(after) > 0 {
		result = append(result, '\n')
		result = append(result, after...)
	}
	return result, nil
}

// findInsertPosition returns the byte offset where to insert the Slices section:
// after the first ## Requirements or ## Acceptance Criteria line, or at end of file.
func findInsertPosition(content []byte) int {
	lines := bytes.Split(content, []byte("\n"))
	for i, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if bytes.HasPrefix(trimmed, []byte("## ")) {
			heading := string(trimmed)
			if heading == requirementsHeading || heading == acceptanceHeading {
				// Insert after this line (include newline)
				pos := 0
				for j := 0; j <= i; j++ {
					pos += len(lines[j])
					if j < len(lines)-1 {
						pos++ // newline
					}
				}
				return pos
			}
		}
	}
	// Insert at end
	return len(content)
}
