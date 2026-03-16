package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"kira/internal/config"
	"kira/internal/roadmap"
	"kira/internal/validation"

	"github.com/spf13/cobra"
)

var roadmapApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Promote ad-hoc roadmap items to work items",
	Long: `Reads ROADMAP.yml, selects entries by optional filters (--period, --workstream,
--owner, --filter path), and promotes ad-hoc items (title, no id) to new work item
files in the backlog. Group metadata is merged (child wins). Use --dry-run to see
what would be promoted without writing.`,
	Args:          cobra.NoArgs,
	RunE:          runRoadmapApply,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	roadmapCmd.AddCommand(roadmapApplyCmd)
	roadmapApplyCmd.Flags().String("period", "", "Filter by period (e.g. Q1-26)")
	roadmapApplyCmd.Flags().String("workstream", "", "Filter by workstream")
	roadmapApplyCmd.Flags().String("owner", "", "Filter by owner")
	roadmapApplyCmd.Flags().String("filter", "", "Hierarchical path filter (e.g. workstream:auth/epic:oauth)")
	roadmapApplyCmd.Flags().Bool("dry-run", false, "Show what would be promoted without making changes")
}

func runRoadmapApply(cmd *cobra.Command, _ []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := checkWorkDir(cfg); err != nil {
		return err
	}

	path := filepath.Join(cfg.ConfigDir, roadmapFilename)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s not found", roadmapFilename)
		}
		return fmt.Errorf("roadmap file: %w", err)
	}
	f, err := roadmap.LoadFile(cfg.ConfigDir, path)
	if err != nil {
		return fmt.Errorf("load roadmap: %w", err)
	}

	flt, err := buildApplyFilter(cmd)
	if err != nil {
		return err
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		return runRoadmapApplyDryRun(f, flt)
	}

	promote := func(mergedMeta map[string]interface{}, title string) (string, error) {
		nextID, err := validation.GetNextID(cfg)
		if err != nil {
			return "", fmt.Errorf("get next ID: %w", err)
		}
		inputs := buildPromoteInputs(mergedMeta, title, nextID)
		if err := writeWorkItemFile(cfg, "task", nextID, title, "backlog", inputs); err != nil {
			return "", err
		}
		return nextID, nil
	}

	errs := roadmap.ApplyPromotions(f, flt, promote)
	for _, e := range errs {
		fmt.Fprintf(os.Stderr, "promote failed: %v\n", e)
	}
	if err := roadmap.SaveFile(cfg.ConfigDir, path, f); err != nil {
		return fmt.Errorf("save roadmap: %w", err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("%d promotion(s) failed", len(errs))
	}
	fmt.Println("Roadmap apply complete.")
	return nil
}

func buildApplyFilter(cmd *cobra.Command) (*roadmap.Filter, error) {
	period, _ := cmd.Flags().GetString("period")
	workstream, _ := cmd.Flags().GetString("workstream")
	owner, _ := cmd.Flags().GetString("owner")
	pathFilter, _ := cmd.Flags().GetString("filter")
	flt := &roadmap.Filter{Period: period, Workstream: workstream, Owner: owner}
	if pathFilter != "" {
		segs, err := roadmap.ParsePathFilter(pathFilter)
		if err != nil {
			return nil, fmt.Errorf("--filter: %w", err)
		}
		flt.Path = segs
	}
	return flt, nil
}

func buildPromoteInputs(mergedMeta map[string]interface{}, title, nextID string) map[string]string {
	inputs := map[string]string{
		"id":      nextID,
		"title":   title,
		"status":  "backlog",
		"created": time.Now().UTC().Format("2006-01-02"),
	}
	if mergedMeta == nil {
		return inputs
	}
	if v, ok := mergedMeta["owner"]; ok {
		if s, ok := v.(string); ok {
			inputs["assigned"] = s
		}
	}
	if v, ok := mergedMeta["tags"]; ok {
		switch t := v.(type) {
		case []interface{}:
			var parts []string
			for _, item := range t {
				if s, ok := item.(string); ok {
					parts = append(parts, s)
				}
			}
			if len(parts) > 0 {
				inputs["tags"] = fmt.Sprintf("%v", parts)
			}
		case []string:
			inputs["tags"] = fmt.Sprintf("%v", t)
		}
	}
	// Optional: map other meta (period, workstream, outcome) into description or leave for later
	return inputs
}

func runRoadmapApplyDryRun(f *roadmap.File, flt *roadmap.Filter) error {
	selected := roadmap.SelectEntries(f, flt)
	var count int
	for _, e := range selected {
		if e.IsAdHoc() {
			count++
			fmt.Printf("Would promote: %q\n", e.Title)
		}
	}
	if count == 0 {
		fmt.Println("No ad-hoc items to promote (filter may exclude all).")
	}
	return nil
}
