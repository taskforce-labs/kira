// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"kira/internal/config"
)

var ideaCmd = &cobra.Command{
	Use:   "idea",
	Short: "Manage ideas in IDEAS.md",
	Long:  `Adds ideas with timestamps to IDEAS.md or lists existing ideas.`,
}

var ideaAddCmd = &cobra.Command{
	Use:   "add <description>",
	Short: "Add an idea to IDEAS.md",
	Long:  `Adds an idea with a timestamp to the IDEAS.md file in numbered format.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		if err := checkWorkDir(cfg); err != nil {
			return err
		}
		description := args[0]
		return addIdeaWithNumber(cfg, description)
	},
}

var ideaListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all ideas",
	Long:  `Lists all ideas from IDEAS.md with their numbers and timestamps.`,
	Args:  cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		if err := checkWorkDir(cfg); err != nil {
			return err
		}
		return listIdeas(cfg)
	},
}

func init() {
	ideaCmd.AddCommand(ideaAddCmd)
	ideaCmd.AddCommand(ideaListCmd)
	// Support legacy "idea <description>" syntax
	ideaCmd.RunE = func(_ *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		if err := checkWorkDir(cfg); err != nil {
			return err
		}
		if len(args) == 0 {
			return ideaCmd.Help()
		}
		return addIdeaWithNumber(cfg, args[0])
	}
	ideaCmd.Args = cobra.MaximumNArgs(1)
}

func addIdeaWithNumber(cfg *config.Config, description string) error {
	// Parse existing file or create new structure
	ideasFile, err := parseIdeasFile(cfg)
	if err != nil {
		// If file doesn't exist, create basic structure
		if strings.Contains(err.Error(), "not found") {
			ideasFile = &IdeasFile{
				BeforeIdeas: `# Ideas

This file is for capturing quick ideas and thoughts that don't fit into formal work items yet.

## How to use
- Add ideas with timestamps using ` + "`kira idea add \"your idea here\"`" + `
- Or manually add entries below

`,
				Ideas:      make(map[int]*Idea),
				IdeaLines:  make(map[int]int),
				Lines:      []string{},
				AfterIdeas: "",
			}
		} else {
			return err
		}
	}

	// Fill gaps in numbering before adding new idea
	if len(ideasFile.Ideas) > 0 {
		if err := fillGapsInIdeas(ideasFile); err != nil {
			return fmt.Errorf("failed to fill gaps in ideas: %w", err)
		}
		// Write back to file after renumbering
		if err := writeIdeasFile(ideasFile, cfg); err != nil {
			return fmt.Errorf("failed to write IDEAS.md after renumbering: %w", err)
		}
		// Re-parse to get updated structure
		ideasFile, err = parseIdeasFile(cfg)
		if err != nil {
			return fmt.Errorf("failed to re-parse IDEAS.md: %w", err)
		}
	}

	// Get next idea number (after gaps are filled)
	nextNumber, err := getNextIdeaNumber(cfg)
	if err != nil {
		return fmt.Errorf("failed to get next idea number: %w", err)
	}

	// Create new idea with date format (not datetime)
	timestamp := time.Now().UTC().Format("2006-01-02")
	newIdea := &Idea{
		Number:    nextNumber,
		Timestamp: timestamp,
		Text:      description,
		LineIndex: len(ideasFile.Lines), // Will be updated when written
	}

	// Add to ideas map
	ideasFile.Ideas[nextNumber] = newIdea

	// Write back to file
	if err := writeIdeasFile(ideasFile, cfg); err != nil {
		return fmt.Errorf("failed to write IDEAS.md: %w", err)
	}

	fmt.Printf("Added idea %d: %s\n", nextNumber, description)
	return nil
}

func listIdeas(cfg *config.Config) error {
	ideasFile, err := parseIdeasFile(cfg)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fmt.Println("No ideas found.")
			return nil
		}
		if strings.Contains(err.Error(), "No ideas found") {
			fmt.Println("No ideas found.")
			return nil
		}
		return err
	}

	if len(ideasFile.Ideas) == 0 {
		fmt.Println("No ideas found.")
		return nil
	}

	// Collect and sort idea numbers
	ideaNumbers := make([]int, 0, len(ideasFile.Ideas))
	for number := range ideasFile.Ideas {
		ideaNumbers = append(ideaNumbers, number)
	}

	// Sort idea numbers
	for i := 1; i < len(ideaNumbers); i++ {
		key := ideaNumbers[i]
		j := i - 1
		for j >= 0 && ideaNumbers[j] > key {
			ideaNumbers[j+1] = ideaNumbers[j]
			j--
		}
		ideaNumbers[j+1] = key
	}

	// Display ideas
	for _, number := range ideaNumbers {
		idea := ideasFile.Ideas[number]
		fmt.Printf("%d. [%s] %s\n", idea.Number, idea.Timestamp, idea.Text)
	}

	return nil
}
