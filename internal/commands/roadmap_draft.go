package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"kira/internal/config"
	"kira/internal/roadmap"

	"github.com/spf13/cobra"
)

var roadmapDraftCmd = &cobra.Command{
	Use:   "draft <name>",
	Short: "Create a draft roadmap from current",
	Long: `Creates ROADMAP-<name>.yml with items from the current roadmap. By default
includes all outstanding items (status not done/released). Use --empty for an
empty draft, or --status/--period/--workstream to filter. Use --include-all to
copy all items regardless of status.`,
	Args:          cobra.ExactArgs(1),
	RunE:          runRoadmapDraft,
	SilenceUsage:  true,
	SilenceErrors: true,
}

var roadmapPromoteCmd = &cobra.Command{
	Use:   "promote <draft-name>",
	Short: "Make a draft the current roadmap and archive the previous",
	Long: `Moves ROADMAP-<draft-name>.yml to ROADMAP.yml (becomes current), archives the
previous ROADMAP.yml to .work/{archived}/roadmap/ROADMAP-{timestamp}.yml, and
commits the rename with git. Prompts for confirmation unless --yes.`,
	Args:          cobra.ExactArgs(1),
	RunE:          runRoadmapPromote,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	roadmapCmd.AddCommand(roadmapDraftCmd)
	roadmapCmd.AddCommand(roadmapPromoteCmd)
	roadmapDraftCmd.Flags().Bool("empty", false, "Create draft with no items")
	roadmapDraftCmd.Flags().String("period", "", "Only include items with this period")
	roadmapDraftCmd.Flags().String("workstream", "", "Only include items in this workstream")
	roadmapDraftCmd.Flags().String("status", "", "Only include items with these statuses (comma-separated)")
	roadmapDraftCmd.Flags().Bool("include-all", false, "Include all items regardless of status")
	roadmapPromoteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
}

func runRoadmapDraft(cmd *cobra.Command, args []string) error {
	name := strings.TrimSpace(args[0])
	if name == "" {
		return fmt.Errorf("draft name is required")
	}
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

	empty, _ := cmd.Flags().GetBool("empty")
	if empty {
		f.Roadmap = nil
	} else {
		statusFilter, _ := cmd.Flags().GetString("status")
		period, _ := cmd.Flags().GetString("period")
		workstream, _ := cmd.Flags().GetString("workstream")
		includeAll, _ := cmd.Flags().GetBool("include-all")
		filter := buildDraftFilter(statusFilter, period, workstream, includeAll)
		resolved := resolveStatusForIDs(f, cfg)
		f.Roadmap = copyDraftEntries(f.Roadmap, filter, resolved)
	}

	draftPath := filepath.Join(cfg.ConfigDir, "ROADMAP-"+name+".yml")
	if err := roadmap.SaveFile(cfg.ConfigDir, draftPath, f); err != nil {
		return fmt.Errorf("save draft: %w", err)
	}
	fmt.Printf("Created draft %s\n", filepath.Base(draftPath))
	return nil
}

type draftFilterCfg struct {
	includeAll  bool
	outstanding bool
	statusSet   map[string]bool
	period      string
	workstream  string
}

func buildDraftFilter(statusFilter, period, workstream string, includeAll bool) func(id, status string, meta map[string]interface{}) bool {
	cfg := draftFilterCfg{
		includeAll:  includeAll,
		outstanding: statusFilter == "" && period == "" && workstream == "",
		period:      period,
		workstream:  workstream,
	}
	if statusFilter != "" {
		cfg.statusSet = make(map[string]bool)
		for _, s := range strings.Split(statusFilter, ",") {
			cfg.statusSet[strings.TrimSpace(s)] = true
		}
	}
	return func(_, status string, meta map[string]interface{}) bool {
		return draftFilterMatch(cfg, status, meta)
	}
}

func draftFilterMatch(cfg draftFilterCfg, status string, meta map[string]interface{}) bool {
	if cfg.includeAll {
		return true
	}
	if cfg.outstanding {
		return status != "done" && status != "released"
	}
	if len(cfg.statusSet) > 0 && !cfg.statusSet[status] {
		return false
	}
	if cfg.period != "" && metaStr(meta, "period") != cfg.period {
		return false
	}
	if cfg.workstream != "" && metaStr(meta, "workstream") != cfg.workstream {
		return false
	}
	return true
}

func metaStr(meta map[string]interface{}, key string) string {
	if meta == nil {
		return ""
	}
	v, _ := meta[key].(string)
	return v
}

func resolveStatusForIDs(f *roadmap.File, cfg *config.Config) map[string]string {
	ids := roadmap.CollectWorkItemIDs(f)
	out := make(map[string]string)
	for _, id := range ids {
		path, err := findWorkItemFile(id, cfg)
		if err != nil {
			out[id] = ""
			continue
		}
		fm, _, err := parseWorkItemFrontMatter(path, cfg)
		if err != nil {
			out[id] = ""
			continue
		}
		s, _ := fm["status"].(string)
		out[id] = s
	}
	return out
}

func copyDraftEntries(
	entries []roadmap.Entry,
	filter func(id, status string, meta map[string]interface{}) bool,
	resolved map[string]string,
) []roadmap.Entry {
	var out []roadmap.Entry
	for _, e := range entries {
		if e.ID != "" {
			status := resolved[e.ID]
			if !filter(e.ID, status, e.Meta) {
				continue
			}
			out = append(out, e)
			continue
		}
		if e.Title != "" {
			out = append(out, e)
			continue
		}
		if e.Group != "" && len(e.Items) > 0 {
			children := copyDraftEntries(e.Items, filter, resolved)
			if len(children) > 0 {
				out = append(out, roadmap.Entry{Group: e.Group, Meta: e.Meta, Items: children})
			}
		}
	}
	return out
}

func runRoadmapPromote(cmd *cobra.Command, args []string) error {
	name := strings.TrimSpace(args[0])
	if name == "" {
		return fmt.Errorf("draft name is required")
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := checkWorkDir(cfg); err != nil {
		return err
	}
	roadmapDir, draftPath, archivePath, err := preparePromotePaths(cfg, name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(draftPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("draft not found: ROADMAP-%s.yml", name)
		}
		return fmt.Errorf("draft file: %w", err)
	}
	yes, _ := cmd.Flags().GetBool("yes")
	if !yes && !promoteConfirm() {
		return fmt.Errorf("aborted")
	}

	if err := archiveCurrentAndWriteDraft(roadmapDir, archivePath, draftPath); err != nil {
		return err
	}
	if err := gitCommitPromote(cfg.ConfigDir, archivePath, roadmapDir, draftPath); err != nil {
		return err
	}
	fmt.Println("Draft promoted to current roadmap; previous archived.")
	return nil
}

func preparePromotePaths(cfg *config.Config, name string) (roadmapDir, draftPath, archivePath string, err error) {
	archivedFolder, ok := cfg.StatusFolders["archived"]
	if !ok || archivedFolder == "" {
		return "", "", "", fmt.Errorf("status_folders.archived not configured")
	}
	workFolder := config.GetWorkFolderPath(cfg)
	roadmapDir = filepath.Join(cfg.ConfigDir, roadmapFilename)
	draftPath = filepath.Join(cfg.ConfigDir, "ROADMAP-"+name+".yml")
	archiveDir := filepath.Join(cfg.ConfigDir, workFolder, archivedFolder, "roadmap")
	if err := os.MkdirAll(archiveDir, 0o700); err != nil {
		return "", "", "", fmt.Errorf("create archive dir: %w", err)
	}
	ts := time.Now().UTC().Format("2006-01-02T150405Z")
	archivePath = filepath.Join(archiveDir, "ROADMAP-"+ts+".yml")
	return roadmapDir, draftPath, archivePath, nil
}

func promoteConfirm() bool {
	fmt.Print("Promote draft to current roadmap? This will archive the current ROADMAP.yml. [y/N]: ")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return false
	}
	return strings.TrimSpace(strings.ToLower(scanner.Text())) == "y"
}

func archiveCurrentAndWriteDraft(roadmapDir, archivePath, draftPath string) error {
	baseDir := filepath.Dir(roadmapDir)
	if _, err := os.Stat(roadmapDir); err == nil {
		data, err := os.ReadFile(roadmapDir) // #nosec G304 - roadmapDir is under ConfigDir
		if err != nil {
			return fmt.Errorf("read current roadmap: %w", err)
		}
		if err := roadmap.ValidateRoadmapPath(baseDir, archivePath); err != nil {
			return fmt.Errorf("invalid archive path: %w", err)
		}
		if err := os.WriteFile(archivePath, data, 0o600); err != nil {
			return fmt.Errorf("write archive: %w", err)
		}
		if err := os.Remove(roadmapDir); err != nil {
			return fmt.Errorf("remove current roadmap: %w", err)
		}
	}
	data, err := os.ReadFile(draftPath) // #nosec G304 - draftPath is under ConfigDir
	if err != nil {
		return fmt.Errorf("read draft: %w", err)
	}
	if err := roadmap.ValidateRoadmapPath(baseDir, roadmapDir); err != nil {
		return fmt.Errorf("invalid roadmap path: %w", err)
	}
	if err := os.WriteFile(roadmapDir, data, 0o600); err != nil {
		return fmt.Errorf("write current roadmap: %w", err)
	}
	if err := os.Remove(draftPath); err != nil {
		return fmt.Errorf("remove draft: %w", err)
	}
	return nil
}

func gitCommitPromote(gitDir, archivePath, roadmapDir, draftPath string) error {
	relArchive, _ := filepath.Rel(gitDir, archivePath)
	relRoadmap, _ := filepath.Rel(gitDir, roadmapDir)
	relDraft, _ := filepath.Rel(gitDir, draftPath)
	// #nosec G204 - paths are from config and roadmap filenames, not user input
	c := exec.Command("git", "add", relArchive, relRoadmap, relDraft)
	c.Dir = gitDir
	if out, err := c.CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %w (%s)", err, string(out))
	}
	c = exec.Command("git", "status", "--porcelain")
	c.Dir = gitDir
	out, _ := c.Output()
	if len(out) == 0 {
		return nil
	}
	c = exec.Command("git", "commit", "-m", "roadmap: promote draft to current, archive previous")
	c.Dir = gitDir
	if out, err := c.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %w (%s)", err, string(out))
	}
	return nil
}
