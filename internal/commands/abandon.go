// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"kira/internal/config"
)

var abandonCmd = &cobra.Command{
	Use:   "abandon <work-item-id|path> [reason|subfolder]",
	Short: "Archive work items and mark them as abandoned",
	Long: `Archives work items and marks them as abandoned.
Updates work item status to "abandoned" before archival.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(_ *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		if err := checkWorkDir(cfg); err != nil {
			return err
		}

		target := args[0]
		var reasonOrSubfolder string
		if len(args) > 1 {
			reasonOrSubfolder = args[1]
		}

		return abandonWorkItems(cfg, target, reasonOrSubfolder)
	},
}

func abandonWorkItems(cfg *config.Config, target, reasonOrSubfolder string) error {
	workItems, sourcePath, err := resolveAbandonTarget(cfg, target, reasonOrSubfolder)
	if err != nil {
		return err
	}

	if len(workItems) == 0 {
		fmt.Println("No work items found to abandon.")
		return nil
	}

	if err := markWorkItemsAbandoned(workItems, reasonOrSubfolder, cfg); err != nil {
		return err
	}

	archivePath, err := archiveWorkItems(workItems, sourcePath, cfg)
	if err != nil {
		return fmt.Errorf("failed to archive work items: %w", err)
	}

	if err := removeAbandonedFiles(workItems); err != nil {
		return err
	}

	fmt.Printf("Abandoned %d work items to %s\n", len(workItems), archivePath)
	return nil
}

func resolveAbandonTarget(cfg *config.Config, target, reasonOrSubfolder string) ([]string, string, error) {
	if isWorkItemID(target) {
		return resolveByID(cfg, target)
	}
	return resolveByPath(cfg, target, reasonOrSubfolder)
}

func resolveByID(cfg *config.Config, target string) ([]string, string, error) {
	workItemPath, err := findWorkItemFile(target, cfg)
	if err != nil {
		return nil, "", err
	}
	return []string{workItemPath}, filepath.Dir(workItemPath), nil
}

func resolveByPath(cfg *config.Config, target, reasonOrSubfolder string) ([]string, string, error) {
	sourcePath, err := buildSourcePath(cfg, target, reasonOrSubfolder)
	if err != nil {
		return nil, "", err
	}

	workItems, err := getWorkItemFiles(sourcePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get work item files: %w", err)
	}
	return workItems, sourcePath, nil
}

func buildSourcePath(cfg *config.Config, target, reasonOrSubfolder string) (string, error) {
	workFolder := config.GetWorkFolderPath(cfg)
	var sourcePath string
	if strings.Contains(target, "/") {
		sourcePath = filepath.Join(workFolder, target)
	} else {
		statusFolder, exists := cfg.StatusFolders[target]
		if !exists {
			return "", fmt.Errorf("invalid status: %s", target)
		}
		sourcePath = filepath.Join(workFolder, statusFolder)
	}

	if reasonOrSubfolder != "" && !strings.Contains(reasonOrSubfolder, " ") {
		sourcePath = filepath.Join(sourcePath, reasonOrSubfolder)
	}
	return sourcePath, nil
}

func markWorkItemsAbandoned(workItems []string, reasonOrSubfolder string, cfg *config.Config) error {
	for _, workItem := range workItems {
		if err := updateWorkItemStatus(workItem, "abandoned", cfg); err != nil {
			return fmt.Errorf("failed to update work item status: %w", err)
		}

		if reasonOrSubfolder != "" && strings.Contains(reasonOrSubfolder, " ") {
			if err := addAbandonmentReason(workItem, reasonOrSubfolder, cfg); err != nil {
				return fmt.Errorf("failed to add abandonment reason: %w", err)
			}
		}
	}
	return nil
}

func removeAbandonedFiles(workItems []string) error {
	for _, workItem := range workItems {
		if err := os.Remove(workItem); err != nil {
			fmt.Printf("Warning: failed to remove %s: %v\n", workItem, err)
		}
	}
	return nil
}

func isWorkItemID(target string) bool {
	// Simple check: if it's a 3-digit number, it's likely an ID
	return len(target) == 3 && target[0] >= '0' && target[0] <= '9'
}

func addAbandonmentReason(filePath, reason string, cfg *config.Config) error {
	content, err := safeReadFile(filePath, cfg)
	if err != nil {
		return err
	}

	// Add abandonment reason to the end of the file
	abandonmentNote := fmt.Sprintf("\n\n## Abandonment\n\n**Reason:** %s\n**Date:** %s\n", reason, time.Now().Format("2006-01-02 15:04:05"))
	newContent := string(content) + abandonmentNote

	return os.WriteFile(filePath, []byte(newContent), 0o600)
}
