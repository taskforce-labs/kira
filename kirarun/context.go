package kirarun

// Context is a read-only view of the host for workflow scripts.
type Context struct {
	Workspace *Workspace
	Log       *Logger
	Skills    *SkillsView
	Commands  *CommandsView
	Run       RunHandle
}

// SkillsView is a read-only snapshot of available skills (host-defined).
type SkillsView struct {
	Names []string
}

// CommandsView is a read-only snapshot of available commands (host-defined).
type CommandsView struct {
	Names []string
}
