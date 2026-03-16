// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"kira/internal/config"
	"kira/internal/roadmap"

	"github.com/spf13/cobra"
)

const roadmapFilename = "ROADMAP.yml"

var roadmapCmd = &cobra.Command{
	Use:   "roadmap",
	Short: "Manage roadmaps (ROADMAP.yml)",
	Long: `Manage the structured roadmap (ROADMAP.yml).

Plan vs roadmap: PLAN.md (under docs folder, e.g. .docs/PLAN.md) is the free-form
planning doc; ROADMAP.yml is the structured derivative. Extract from product docs
into PLAN.md, then generate or edit ROADMAP.yml. Use roadmap lint to validate refs
and schema; roadmap apply to promote ad-hoc items to work items; roadmap draft and
promote for draft workflows.`,
}

var roadmapLintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Validate the current roadmap file",
	Long: `Validates ROADMAP.yml: schema (each entry has id or title or group+items),
references (every id references an existing work item file), optional stage
allowlist, and optional dependency checks (unknown depends_on, cycles).
Use --check-adhoc to list ad-hoc items.`,
	Args:          cobra.NoArgs,
	RunE:          runRoadmapLint,
	SilenceUsage:  true,
	SilenceErrors: true,
}

var roadmapPlanToRoadmapCmd = &cobra.Command{
	Use:   "plan-to-roadmap",
	Short: "Generate or update ROADMAP.yml from PLAN.md (v1 stub)",
	Long: `Read PLAN.md (from docs folder, e.g. .docs/PLAN.md) and emit or update ROADMAP.yml.
In v1 this is a stub: edit ROADMAP.yml manually or use an external tool/LLM to extract
structure from PLAN.md.`,
	Args: cobra.NoArgs,
	RunE: runRoadmapPlanToRoadmap,
}

func init() {
	roadmapCmd.AddCommand(roadmapLintCmd)
	roadmapCmd.AddCommand(roadmapPlanToRoadmapCmd)
	roadmapLintCmd.Flags().Bool("check-adhoc", false, "List ad-hoc items (entries with title but no id)")
	roadmapLintCmd.Flags().Bool("check-deps", false, "Warn on unknown depends_on IDs and report dependency cycles")
}

func runRoadmapPlanToRoadmap(*cobra.Command, []string) error {
	fmt.Println("plan-to-roadmap: v1 stub. Edit ROADMAP.yml manually or use an external tool to generate from PLAN.md.")
	return nil
}

func runRoadmapLint(cmd *cobra.Command, _ []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := checkWorkDir(cfg); err != nil {
		return err
	}

	path := filepath.Join(cfg.ConfigDir, roadmapFilename)
	f, err := loadRoadmapForLint(cfg.ConfigDir, path)
	if err != nil {
		return err
	}

	if err := roadmapLintSchema(f); err != nil {
		return err
	}
	if err := roadmapLintRefs(f, cfg); err != nil {
		return err
	}
	roadmapLintDeps(cmd, f)
	checkAdhoc, _ := cmd.Flags().GetBool("check-adhoc")
	roadmapLintAdhoc(cmd, f)
	if !checkAdhoc {
		fmt.Println("Roadmap is valid.")
	}
	return nil
}

func loadRoadmapForLint(configDir, path string) (*roadmap.File, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%s not found", roadmapFilename)
		}
		return nil, fmt.Errorf("roadmap file: %w", err)
	}
	f, err := roadmap.LoadFile(configDir, path)
	if err != nil {
		return nil, fmt.Errorf("load roadmap: %w", err)
	}
	return f, nil
}

func roadmapLintSchema(f *roadmap.File) error {
	schemaErrs := roadmap.Validate(f)
	if !roadmap.HasErrors(schemaErrs) {
		return nil
	}
	for _, e := range schemaErrs {
		fmt.Fprintf(os.Stderr, "%s\n", e.Error())
	}
	return fmt.Errorf("schema validation failed")
}

func roadmapLintRefs(f *roadmap.File, cfg *config.Config) error {
	ids := roadmap.CollectWorkItemIDs(f)
	var broken []string
	for _, id := range ids {
		if _, err := findWorkItemFile(id, cfg); err != nil {
			broken = append(broken, id)
		}
	}
	if len(broken) == 0 {
		return nil
	}
	for _, id := range broken {
		fmt.Fprintf(os.Stderr, "warning: work item %s not found (no file with id in front matter)\n", id)
	}
	return fmt.Errorf("broken references: %v", broken)
}

func roadmapLintDeps(cmd *cobra.Command, f *roadmap.File) {
	checkDeps, _ := cmd.Flags().GetBool("check-deps")
	if !checkDeps {
		return
	}
	ids := roadmap.CollectWorkItemIDs(f)
	knownIDs := make(map[string]bool)
	for _, id := range ids {
		knownIDs[id] = true
	}
	refs := roadmap.CollectDependsOn(f)
	for _, r := range refs {
		for _, d := range r.DependsOn {
			if !knownIDs[d] {
				fmt.Fprintf(os.Stderr, "warning: %s depends_on unknown id %s\n", r.ID, d)
			}
		}
	}
	cycle := roadmap.DependencyCycle(f, knownIDs)
	if len(cycle) > 0 {
		fmt.Fprintf(os.Stderr, "warning: dependency cycle: %v\n", cycle)
	}
}

func roadmapLintAdhoc(cmd *cobra.Command, f *roadmap.File) {
	checkAdhoc, _ := cmd.Flags().GetBool("check-adhoc")
	if !checkAdhoc {
		return
	}
	adhoc := roadmap.CollectAdHoc(f)
	if len(adhoc) == 0 {
		fmt.Println("No ad-hoc items.")
		return
	}
	fmt.Printf("Ad-hoc items (%d):\n", len(adhoc))
	for _, a := range adhoc {
		fmt.Printf("  %s: %q\n", a.Path, a.Title)
	}
}
