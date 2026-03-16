package roadmap

// PromoteFunc is called for each ad-hoc entry to promote. It returns the new work item ID or an error.
// On error the entry is not replaced (best-effort: caller may continue with others).
type PromoteFunc func(mergedMeta map[string]interface{}, title string) (newID string, err error)

// ApplyPromotions walks the tree and promotes each ad-hoc entry that matches the filter.
// For each such entry it calls promote(mergedMeta, title); on success replaces the entry with the new ID.
// Failures are collected and returned; promotions continue (best-effort).
func ApplyPromotions(f *File, flt *Filter, promote PromoteFunc) []error {
	if flt == nil {
		flt = &Filter{}
	}
	var errs []error
	walkApply(f.Roadmap, nil, flt, promote, &errs)
	return errs
}

func walkApply(
	entries []Entry, ancestors []map[string]interface{}, flt *Filter, promote PromoteFunc, errs *[]error,
) {
	for i := range entries {
		e := &entries[i]
		meta := e.Meta
		if meta == nil {
			meta = make(map[string]interface{})
		}
		merged := mergeMeta(ancestors, meta)
		path := BuildPath(ancestors, meta, flt.Path)
		pathOK := PathMatches(path, flt.Path)
		metaOK := flt.MatchMeta(merged)
		matches := pathOK && (metaOK || (flt.Period == "" && flt.Workstream == "" && flt.Owner == ""))

		if matches && e.IsAdHoc() {
			newID, err := promote(merged, e.Title)
			if err != nil {
				*errs = append(*errs, err)
			} else {
				e.ID = newID
				e.Title = ""
			}
		}

		ancestorsNext := append(append([]map[string]interface{}(nil), ancestors...), meta)
		if len(e.Items) > 0 {
			walkApply(e.Items, ancestorsNext, flt, promote, errs)
		}
	}
}
