package roadmap

import "strconv"

// CollectWorkItemIDs returns all work item IDs referenced in the roadmap (id fields).
func CollectWorkItemIDs(f *File) []string {
	var ids []string
	walkEntries(f.Roadmap, func(_ string, e *Entry) {
		if e.ID != "" {
			ids = append(ids, e.ID)
		}
	}, "")
	return ids
}

// AdHocRef describes an ad-hoc entry for reporting.
type AdHocRef struct {
	Path  string
	Title string
}

// CollectAdHoc returns all ad-hoc entries (title, no id) in the roadmap.
func CollectAdHoc(f *File) []AdHocRef {
	var refs []AdHocRef
	walkEntries(f.Roadmap, func(path string, e *Entry) {
		if e.IsAdHoc() {
			refs = append(refs, AdHocRef{Path: path, Title: e.Title})
		}
	}, "")
	return refs
}

// DependsOnRef holds an entry path, its id, and the depends_on list from meta.
type DependsOnRef struct {
	Path      string
	ID        string
	DependsOn []string
}

// CollectDependsOn returns all depends_on metadata from entries that have an id.
func CollectDependsOn(f *File) []DependsOnRef {
	var refs []DependsOnRef
	walkEntries(f.Roadmap, func(path string, e *Entry) {
		if e.ID == "" {
			return
		}
		dep := getDependsOn(e.Meta)
		if len(dep) > 0 {
			refs = append(refs, DependsOnRef{Path: path, ID: e.ID, DependsOn: dep})
		}
	}, "")
	return refs
}

func getDependsOn(meta map[string]interface{}) []string {
	if meta == nil {
		return nil
	}
	v, ok := meta["depends_on"]
	if !ok {
		return nil
	}
	switch t := v.(type) {
	case []interface{}:
		var out []string
		for _, item := range t {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return t
	default:
		return nil
	}
}

func walkEntries(entries []Entry, fn func(path string, e *Entry), prefix string) {
	for i := range entries {
		e := &entries[i]
		var path string
		if prefix == "" {
			path = "roadmap[" + strconv.Itoa(i) + "]"
		} else {
			path = prefix + ".items[" + strconv.Itoa(i) + "]"
		}
		fn(path, e)
		if len(e.Items) > 0 {
			walkEntries(e.Items, fn, path)
		}
	}
}

// DependencyCycle returns a cycle of IDs if the roadmap's depends_on graph has a cycle (only among known IDs).
// knownIDs is the set of work item IDs that exist (e.g. resolved via findWorkItemFile).
func DependencyCycle(f *File, knownIDs map[string]bool) []string {
	refs := CollectDependsOn(f)
	// Build adjacency: id -> list of ids it depends on (only include known IDs)
	adj := make(map[string][]string)
	for _, r := range refs {
		if !knownIDs[r.ID] {
			continue
		}
		for _, d := range r.DependsOn {
			if knownIDs[d] {
				adj[r.ID] = append(adj[r.ID], d)
			}
		}
	}
	// DFS to find cycle
	visited := make(map[string]bool)
	inStack := make(map[string]bool)
	path := make([]string, 0, 8)
	var cycle []string
	var visit func(id string) bool
	visit = func(id string) bool {
		visited[id] = true
		inStack[id] = true
		path = append(path, id)
		for _, next := range adj[id] {
			if !visited[next] {
				if visit(next) {
					return true
				}
			} else if inStack[next] {
				// Found cycle: from next back to next
				i := 0
				for path[i] != next {
					i++
				}
				cycle = make([]string, 0, len(path)-i+1)
				for j := i; j < len(path); j++ {
					cycle = append(cycle, path[j])
				}
				cycle = append(cycle, next)
				return true
			}
		}
		path = path[:len(path)-1]
		inStack[id] = false
		return false
	}
	for id := range adj {
		if !visited[id] && visit(id) {
			return cycle
		}
	}
	return nil
}
