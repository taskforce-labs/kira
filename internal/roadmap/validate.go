package roadmap

import (
	"fmt"
	"strings"
)

// ValidateError represents a schema validation error at a path in the roadmap tree.
type ValidateError struct {
	Path    string // e.g. "roadmap[0]", "roadmap[0].items[1]"
	Message string
}

func (e ValidateError) Error() string {
	return e.Path + ": " + e.Message
}

// Validate runs schema validation on the roadmap. Returns all validation errors.
// Each entry must have exactly one of: id (work item ref), title (ad-hoc), or group+items (group).
// Empty or invalid entries are rejected.
func Validate(f *File) []ValidateError {
	var errs []ValidateError
	if f == nil {
		return append(errs, ValidateError{Path: "roadmap", Message: "roadmap is nil"})
	}
	for i := range f.Roadmap {
		validateEntry(&f.Roadmap[i], "roadmap["+fmt.Sprint(i)+"]", &errs)
	}
	return errs
}

func validateEntry(e *Entry, path string, errs *[]ValidateError) {
	hasID := strings.TrimSpace(e.ID) != ""
	hasTitle := strings.TrimSpace(e.Title) != ""
	hasGroup := strings.TrimSpace(e.Group) != ""
	hasItems := len(e.Items) > 0

	appendEmptyOrStructureErrs(path, hasID, hasTitle, hasGroup, hasItems, errs)
	appendMultipleKindErrs(path, hasID, hasTitle, hasGroup, errs)

	for j := range e.Items {
		validateEntry(&e.Items[j], path+".items["+fmt.Sprint(j)+"]", errs)
	}
}

func appendEmptyOrStructureErrs(path string, hasID, hasTitle, hasGroup, hasItems bool, errs *[]ValidateError) {
	if hasID || hasTitle || (hasGroup && hasItems) {
		return
	}
	if !hasID && !hasTitle && !hasGroup && !hasItems {
		*errs = append(*errs, ValidateError{Path: path, Message: "entry is empty: must have id, title, or group+items"})
		return
	}
	if hasGroup && !hasItems {
		*errs = append(*errs, ValidateError{Path: path, Message: "group must have items"})
		return
	}
	if hasItems && !hasGroup && !hasID && !hasTitle {
		*errs = append(*errs, ValidateError{Path: path, Message: "entry with items must have group, id, or title"})
	}
}

func appendMultipleKindErrs(path string, hasID, hasTitle, hasGroup bool, errs *[]ValidateError) {
	if hasID && hasTitle {
		*errs = append(*errs, ValidateError{Path: path, Message: "entry cannot have both id and title"})
	}
	if hasID && hasGroup {
		*errs = append(*errs, ValidateError{Path: path, Message: "entry cannot have both id and group"})
	}
	if hasTitle && hasGroup {
		*errs = append(*errs, ValidateError{Path: path, Message: "entry cannot have both title and group"})
	}
}

// HasErrors returns true if any validation errors are present.
func HasErrors(errs []ValidateError) bool {
	return len(errs) > 0
}
