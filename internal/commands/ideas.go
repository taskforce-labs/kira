// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"kira/internal/config"
)

// Idea represents a parsed idea from IDEAS.md
type Idea struct {
	Number    int
	Timestamp string
	Text      string
	LineIndex int // Original line index in file for renumbering
}

// IdeasFile represents the parsed structure of IDEAS.md
type IdeasFile struct {
	BeforeIdeas string        // Content before ## List header
	Ideas       map[int]*Idea // Map of idea number to Idea
	AfterIdeas  string        // Content after ideas section (if any)
	Lines       []string      // All lines for reconstruction
	IdeaLines   map[int]int   // Map of idea number to line index
}

// parseMultiLineIdea parses continuation lines for a multi-line idea
func parseMultiLineIdea(lines []string, startIndex int, initialText string, ideaRegex *regexp.Regexp) (string, int) {
	text := initialText
	i := startIndex

	for j := i + 1; j < len(lines); j++ {
		nextLine := lines[j]
		nextTrimmed := strings.TrimSpace(nextLine)

		// Stop if we hit another numbered idea, header, or blank line followed by non-indented content
		if nextTrimmed == "" {
			// Blank line - check if next non-blank line is a new idea or header
			if j+1 < len(lines) {
				nextNonBlank := strings.TrimSpace(lines[j+1])
				if strings.HasPrefix(nextNonBlank, "#") || ideaRegex.MatchString(lines[j+1]) {
					break
				}
			}
			// Include blank line as part of idea
			text += "\n" + nextLine
			i = j
			continue
		}

		if strings.HasPrefix(nextTrimmed, "#") {
			break
		}

		// If next line starts with a number and dot, it's a new idea
		if ideaRegex.MatchString(nextLine) {
			break
		}

		// If line is indented or continues the idea, include it
		if strings.HasPrefix(nextLine, " ") || strings.HasPrefix(nextLine, "\t") || nextTrimmed != "" {
			text += "\n" + nextLine
			i = j
		} else {
			break
		}
	}

	return text, i
}

// parseIdeasFile reads and parses IDEAS.md, extracting ideas under the ## List header
func parseIdeasFile(cfg *config.Config) (*IdeasFile, error) {
	workFolder := config.GetWorkFolderPath(cfg)
	ideasPath := filepath.Join(workFolder, "IDEAS.md")

	content, err := safeReadFile(ideasPath, cfg)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("IDEAS.md not found. Run 'kira init' first")
		}
		return nil, fmt.Errorf("failed to read IDEAS.md: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	result := &IdeasFile{
		Ideas:     make(map[int]*Idea),
		IdeaLines: make(map[int]int),
		Lines:     lines,
	}

	// Find the ## List header
	ideasHeaderIndex := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "## List" {
			ideasHeaderIndex = i
			break
		}
	}

	if ideasHeaderIndex == -1 {
		return nil, fmt.Errorf("IDEAS.md is missing the '## List' header")
	}

	// Store content before ## List header
	if ideasHeaderIndex > 0 {
		result.BeforeIdeas = strings.Join(lines[:ideasHeaderIndex], "\n")
	}

	// Parse ideas after the header
	ideaRegex := regexp.MustCompile(`^(\d+)\.\s+\[([^\]]+)\]\s+(.+)$`)

	for i := ideasHeaderIndex + 1; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Check if we've left the ideas section (hit another header)
		if strings.HasPrefix(trimmed, "#") && trimmed != "## List" {
			result.AfterIdeas = strings.Join(lines[i:], "\n")
			break
		}

		// Try to match numbered idea format: "1. [timestamp] text"
		matches := ideaRegex.FindStringSubmatch(line)
		if matches != nil {
			number, err := strconv.Atoi(matches[1])
			if err != nil {
				continue // Skip malformed numbers
			}

			timestamp := matches[2]
			text := matches[3]

			// Check for multi-line idea (next lines indented)
			text, i = parseMultiLineIdea(lines, i, text, ideaRegex)

			// Trim trailing newlines from idea text
			text = strings.TrimRight(text, "\n")

			result.Ideas[number] = &Idea{
				Number:    number,
				Timestamp: timestamp,
				Text:      text,
				LineIndex: i,
			}
			result.IdeaLines[number] = i
		}
	}

	// If no ideas found and we're in the ideas section
	// This is OK - ideas section can be empty (e.g., after all ideas are converted)
	// Empty section is fine, so we don't return an error

	return result, nil
}

// getIdeaByNumber retrieves a specific idea by number
func getIdeaByNumber(number int, cfg *config.Config) (*Idea, error) {
	ideasFile, err := parseIdeasFile(cfg)
	if err != nil {
		return nil, err
	}

	idea, exists := ideasFile.Ideas[number]
	if !exists {
		return nil, fmt.Errorf("Idea %d not found", number)
	}

	return idea, nil
}

// getNextIdeaNumber finds the highest numbered idea and returns the next number
func getNextIdeaNumber(cfg *config.Config) (int, error) {
	ideasFile, err := parseIdeasFile(cfg)
	if err != nil {
		// If file doesn't exist or has no ideas, start at 1
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "No ideas found") {
			return 1, nil
		}
		return 0, err
	}

	maxNumber := 0
	for number := range ideasFile.Ideas {
		if number > maxNumber {
			maxNumber = number
		}
	}

	return maxNumber + 1, nil
}

// writeIdeasFile writes ideas back to IDEAS.md preserving file structure
func writeIdeasFile(ideasFile *IdeasFile, cfg *config.Config) error {
	workFolder := config.GetWorkFolderPath(cfg)
	ideasPath := filepath.Join(workFolder, "IDEAS.md")

	var result strings.Builder

	// Write content before ## List header
	if ideasFile.BeforeIdeas != "" {
		result.WriteString(ideasFile.BeforeIdeas)
		result.WriteString("\n")
	}

	// Write ## List header
	result.WriteString("## List\n")

	// Write ideas in order (sorted by line index to preserve file order)
	// Collect ideas with their line indices
	type ideaWithLine struct {
		idea *Idea
		line int
	}
	ideasWithLines := make([]ideaWithLine, 0, len(ideasFile.Ideas))
	for _, idea := range ideasFile.Ideas {
		ideasWithLines = append(ideasWithLines, ideaWithLine{idea: idea, line: idea.LineIndex})
	}

	// Sort by line index
	for i := 1; i < len(ideasWithLines); i++ {
		key := ideasWithLines[i]
		j := i - 1
		for j >= 0 && ideasWithLines[j].line > key.line {
			ideasWithLines[j+1] = ideasWithLines[j]
			j--
		}
		ideasWithLines[j+1] = key
	}

	// Write ideas in file order
	for _, item := range ideasWithLines {
		idea := item.idea
		ideaLine := fmt.Sprintf("%d. [%s] %s", idea.Number, idea.Timestamp, idea.Text)
		result.WriteString(ideaLine)
		result.WriteString("\n")
	}

	// Write content after ideas section
	if ideasFile.AfterIdeas != "" {
		result.WriteString("\n")
		result.WriteString(ideasFile.AfterIdeas)
	}

	// Write to file
	if err := os.WriteFile(ideasPath, []byte(result.String()), 0o600); err != nil {
		return fmt.Errorf("failed to write IDEAS.md: %w", err)
	}

	return nil
}

// fillGapsInIdeas renumbers ideas to fill gaps, ensuring sequential numbering (1, 2, 3...)
// It preserves the order of ideas in the file but fills gaps in numbering
func fillGapsInIdeas(ideasFile *IdeasFile) error {
	if len(ideasFile.Ideas) == 0 {
		return nil
	}

	// Collect ideas sorted by their line index (preserve file order)
	type ideaWithIndex struct {
		idea      *Idea
		index     int
		oldNumber int
	}

	ideasList := make([]ideaWithIndex, 0, len(ideasFile.Ideas))
	for _, idea := range ideasFile.Ideas {
		ideasList = append(ideasList, ideaWithIndex{
			idea:      idea,
			index:     idea.LineIndex,
			oldNumber: idea.Number,
		})
	}

	// Sort by line index to preserve file order
	for i := 1; i < len(ideasList); i++ {
		key := ideasList[i]
		j := i - 1
		for j >= 0 && ideasList[j].index > key.index {
			ideasList[j+1] = ideasList[j]
			j--
		}
		ideasList[j+1] = key
	}

	// Find the highest sequential number by checking actual idea numbers
	// We need to find the largest N where ideas 1..N all exist
	ideaNumbers := make(map[int]bool)
	for _, item := range ideasList {
		ideaNumbers[item.oldNumber] = true
	}

	highestSequential := 0
	for i := 1; i <= len(ideasList); i++ {
		if ideaNumbers[i] {
			highestSequential = i
		} else {
			// Found a gap, stop here
			break
		}
	}

	// Renumber ideas sequentially, preserving file order
	newIdeas := make(map[int]*Idea)
	newIdeaLines := make(map[int]int)
	nextNumber := 1

	for _, item := range ideasList {
		// If this idea is within the sequential block, keep its number if it matches
		// Otherwise, assign the next sequential number
		if item.oldNumber <= highestSequential && item.oldNumber == nextNumber {
			// This idea is already correctly numbered
			newIdeas[item.oldNumber] = item.idea
			newIdeaLines[item.oldNumber] = item.index
			nextNumber = item.oldNumber + 1
		} else {
			// This idea needs renumbering to fill gaps
			item.idea.Number = nextNumber
			newIdeas[nextNumber] = item.idea
			newIdeaLines[nextNumber] = item.index
			nextNumber++
		}
	}

	ideasFile.Ideas = newIdeas
	ideasFile.IdeaLines = newIdeaLines

	return nil
}

// renumberIdeas renumbers ideas sequentially (1, 2, 3...) preserving order
func renumberIdeas(ideasFile *IdeasFile) error {
	// Collect ideas sorted by their original line index
	type ideaWithIndex struct {
		idea  *Idea
		index int
	}

	ideasList := make([]ideaWithIndex, 0, len(ideasFile.Ideas))
	for _, idea := range ideasFile.Ideas {
		ideasList = append(ideasList, ideaWithIndex{idea: idea, index: idea.LineIndex})
	}

	// Sort by line index
	for i := 1; i < len(ideasList); i++ {
		key := ideasList[i]
		j := i - 1
		for j >= 0 && ideasList[j].index > key.index {
			ideasList[j+1] = ideasList[j]
			j--
		}
		ideasList[j+1] = key
	}

	// Renumber sequentially
	newIdeas := make(map[int]*Idea)
	newIdeaLines := make(map[int]int)
	for i, item := range ideasList {
		newNumber := i + 1
		item.idea.Number = newNumber
		newIdeas[newNumber] = item.idea
		newIdeaLines[newNumber] = item.index
	}

	ideasFile.Ideas = newIdeas
	ideasFile.IdeaLines = newIdeaLines

	return nil
}

// IdeaTitleDescription represents the parsed title and description from an idea
type IdeaTitleDescription struct {
	Title       string
	Description string
}

// parseIdeaTitleDescription extracts title and description from idea text per PRD rules
func parseIdeaTitleDescription(ideaText string) IdeaTitleDescription {
	ideaText = strings.TrimSpace(ideaText)
	result := IdeaTitleDescription{}

	// Check if idea contains a colon
	colonIndex := strings.Index(ideaText, ":")
	if colonIndex != -1 {
		// Has colon: text before colon = title, after = description
		result.Title = strings.TrimSpace(ideaText[:colonIndex])
		result.Description = strings.TrimSpace(ideaText[colonIndex+1:])
	} else {
		// No colon: split by words
		words := strings.Fields(ideaText)
		wordCount := len(words)

		if wordCount < 5 {
			// Fewer than 5 words: entire text as title, empty description
			result.Title = ideaText
			result.Description = ""
		} else {
			// 5 or more words: first 5 words as title, entire text as description
			result.Title = strings.Join(words[:5], " ")
			result.Description = ideaText
		}
	}

	// If title is empty after parsing, use first 5 words of description as title
	if result.Title == "" && result.Description != "" {
		words := strings.Fields(result.Description)
		if len(words) >= 5 {
			result.Title = strings.Join(words[:5], " ")
		} else {
			result.Title = result.Description
		}
	}

	// Trim whitespace from both
	result.Title = strings.TrimSpace(result.Title)
	result.Description = strings.TrimSpace(result.Description)

	return result
}

// removeIdeaByNumber removes an idea by number and renumbers remaining ideas
func removeIdeaByNumber(number int, cfg *config.Config) error {
	ideasFile, err := parseIdeasFile(cfg)
	if err != nil {
		return err
	}

	// Check if idea exists
	if _, exists := ideasFile.Ideas[number]; !exists {
		return fmt.Errorf("Idea %d not found", number)
	}

	// Remove the idea
	delete(ideasFile.Ideas, number)
	delete(ideasFile.IdeaLines, number)

	// Renumber remaining ideas
	if err := renumberIdeas(ideasFile); err != nil {
		return fmt.Errorf("failed to renumber ideas: %w", err)
	}

	// Write back to file
	if err := writeIdeasFile(ideasFile, cfg); err != nil {
		return fmt.Errorf("failed to write IDEAS.md: %w", err)
	}

	return nil
}
