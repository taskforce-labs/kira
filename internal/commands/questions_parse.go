package commands

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

const (
	questionsHeading        = "## Questions"
	questionsOptionsHeading = "#### Options"
)

var (
	questionHeadingRegex = regexp.MustCompile(`^###\s+(\d+)\.\s*(.*)$`)
	checkboxCheckedLine  = regexp.MustCompile(`^\s*-\s+\[[xX]\]\s*`)
)

// QuestionEntry is one ### N. heading under ## Questions with parse metadata.
type QuestionEntry struct {
	Title     string
	Line      int // 1-based line number of the ### heading in the file
	Answered  bool
	BodyLines []string // non-heading lines in the subsection (for display / future use)
}

// findQuestionsSection returns start and end byte offsets of the ## Questions section (up to next ## at column start, not inside code fences).
func findQuestionsSection(content []byte) (start, end int, found bool) {
	sectionStart := -1
	inCodeBlock := false
	pos := 0
	for pos < len(content) {
		lineStart, trimmedStr, nextPos := nextLine(content, pos)
		pos = nextPos

		inCodeBlock = updateCodeBlockState(inCodeBlock, trimmedStr)
		if inCodeBlock {
			continue
		}

		if trimmedStr == questionsHeading {
			if sectionStart < 0 {
				sectionStart = lineStart
			}
			continue // do not treat this line as "next ##" end marker
		}
		if sectionStart >= 0 && strings.HasPrefix(trimmedStr, "## ") {
			end = trimSectionEnd(content, sectionStart, lineStart)
			return sectionStart, end, true
		}
	}
	if sectionStart >= 0 {
		end = trimSectionEnd(content, sectionStart, len(content))
		return sectionStart, end, true
	}
	return 0, 0, false
}

// ParseQuestionsFromMarkdown extracts questions from a full file body; line numbers are 1-based in the file.
func ParseQuestionsFromMarkdown(content []byte) []QuestionEntry {
	secStart, secEnd, ok := findQuestionsSection(content)
	if !ok {
		return nil
	}
	section := content[secStart:secEnd]
	lines := strings.Split(string(section), "\n")

	lineNumBase := bytes.Count(content[:secStart], []byte{'\n'})

	var out []QuestionEntry
	var cur *QuestionEntry
	flush := func() {
		if cur != nil {
			out = append(out, *cur)
			cur = nil
		}
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		fileLine := lineNumBase + i + 1

		if strings.HasPrefix(trimmed, "### ") {
			flush()
			m := questionHeadingRegex.FindStringSubmatch(trimmed)
			if len(m) < 3 {
				continue
			}
			title := strings.TrimSpace(m[2])
			cur = &QuestionEntry{Title: title, Line: fileLine, BodyLines: nil}
			continue
		}
		if cur == nil {
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			continue
		}

		cur.BodyLines = append(cur.BodyLines, line)
	}
	flush()

	for i := range out {
		out[i].Answered = questionAnswered(out[i].BodyLines)
	}
	return out
}

// questionAnswered returns true if there is a #### Options block with at least one checked checkbox line.
func questionAnswered(bodyLines []string) bool {
	inOptions := false
	for _, line := range bodyLines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "### ") {
			break
		}
		if strings.HasPrefix(trimmed, "## ") {
			break
		}
		if trimmed == questionsOptionsHeading {
			inOptions = true
			continue
		}
		if inOptions {
			if strings.HasPrefix(trimmed, "#### ") && trimmed != questionsOptionsHeading {
				break
			}
			if strings.HasPrefix(trimmed, "### ") {
				break
			}
			if checkboxCheckedLine.MatchString(trimmed) {
				return true
			}
		}
	}
	return false
}

// UnansweredQuestions returns entries that have no #### Options or no checked option.
func UnansweredQuestions(entries []QuestionEntry) []QuestionEntry {
	var u []QuestionEntry
	for _, e := range entries {
		if !e.Answered {
			u = append(u, e)
		}
	}
	return u
}

// QuestionDisplayText returns title plus optional first line of body for context.
func QuestionDisplayText(e QuestionEntry) string {
	if len(e.BodyLines) == 0 {
		return e.Title
	}
	for _, ln := range e.BodyLines {
		t := strings.TrimSpace(ln)
		if t == "" || t == questionsOptionsHeading {
			continue
		}
		if strings.HasPrefix(t, "#### ") {
			continue
		}
		if strings.HasPrefix(t, "### ") {
			break
		}
		return fmt.Sprintf("%s — %s", e.Title, t)
	}
	return e.Title
}
