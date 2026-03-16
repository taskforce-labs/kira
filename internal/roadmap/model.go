// Package roadmap provides the data model and operations for Kira roadmaps.
package roadmap

// File is the top-level structure for ROADMAP.yml (key: "roadmap").
type File struct {
	Roadmap []Entry `yaml:"roadmap"`
}

// Entry represents a single roadmap entry: work item ref, ad-hoc item, or group.
// Exactly one of ID, Title, or Group+Items is set per canonical form.
type Entry struct {
	// ID is set for work item references.
	ID string `yaml:"id,omitempty"`
	// Title is set for ad-hoc items.
	Title string `yaml:"title,omitempty"`
	// Group is the group title when this entry is a container (nested items).
	Group string  `yaml:"group,omitempty"`
	Items []Entry `yaml:"items,omitempty"`
	// Meta holds optional metadata (period, workstream, owner, tags, depends_on, etc.).
	Meta map[string]interface{} `yaml:"meta,omitempty"`
}

// IsWorkItemRef returns true if the entry is a reference to a work item by ID.
func (e *Entry) IsWorkItemRef() bool {
	return e.ID != ""
}

// IsAdHoc returns true if the entry is an ad-hoc item (title, no ID).
func (e *Entry) IsAdHoc() bool {
	return e.Title != "" && e.ID == ""
}

// IsGroup returns true if the entry is a group container.
func (e *Entry) IsGroup() bool {
	return e.Group != "" && len(e.Items) > 0
}

// Kind returns the entry kind for validation: "id", "title", or "group".
func (e *Entry) Kind() string {
	if e.ID != "" {
		return "id"
	}
	if e.Title != "" {
		return "title"
	}
	if e.Group != "" || len(e.Items) > 0 {
		return "group"
	}
	return ""
}
