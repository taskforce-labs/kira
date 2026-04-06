package kirarun

// Workspace exposes read-only paths for the kira project.
type Workspace struct {
	root string
}

// Root returns the absolute project root (directory containing kira.yml).
func (w *Workspace) Root() string {
	if w == nil {
		return ""
	}
	return w.root
}

// NewWorkspace returns a read-only workspace view. root must be absolute.
func NewWorkspace(root string) *Workspace {
	return &Workspace{root: root}
}
