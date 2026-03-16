package roadmap

import (
	"strings"
	"unicode"
)

// Filter holds optional meta filters and an optional hierarchical path filter.
type Filter struct {
	Period     string // if set, entry or ancestor must have meta.period == Period
	Workstream string
	Owner      string
	Path       []PathSegment // if non-empty, path-from-root (by meta) must match prefix or full
}

// PathSegment is one segment of a hierarchical path filter (key:value).
type PathSegment struct {
	Key   string
	Value string
}

// ParsePathFilter parses a hierarchical path string into segments.
// Format: key:value/key:value. Values containing : or / may be quoted: key:"value" or key:'value'.
// Unquoted values are restricted to alphanumeric, hyphen, underscore.
func ParsePathFilter(s string) ([]PathSegment, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	var segments []PathSegment
	for {
		seg, rest, err := parseNextSegment(s)
		if err != nil {
			return nil, err
		}
		if seg != nil {
			segments = append(segments, *seg)
		}
		s = rest
		if s == "" {
			break
		}
		if s[0] != '/' {
			return nil, &pathParseError{msg: "expected / after segment"}
		}
		s = strings.TrimLeft(s[1:], " ")
	}
	return segments, nil
}

type pathParseError struct{ msg string }

func (e *pathParseError) Error() string { return e.msg }

func parseNextSegment(s string) (seg *PathSegment, rest string, err error) {
	key, s, err := parsePathKey(s)
	if err != nil {
		return nil, "", err
	}
	s = strings.TrimLeft(s, " ")
	if len(s) == 0 || s[0] != ':' {
		return nil, "", &pathParseError{msg: "expected : after key"}
	}
	s = strings.TrimLeft(s[1:], " ")
	if len(s) == 0 {
		return nil, "", &pathParseError{msg: "missing value"}
	}
	value, rest, err := parsePathValue(s)
	if err != nil {
		return nil, "", err
	}
	return &PathSegment{Key: key, Value: value}, rest, nil
}

func parsePathKey(s string) (key, rest string, err error) {
	i := 0
	for i < len(s) && (unicode.IsLetter(rune(s[i])) || unicode.IsDigit(rune(s[i])) || s[i] == '_') {
		i++
	}
	if i == 0 {
		return "", "", &pathParseError{msg: "missing key"}
	}
	return s[:i], strings.TrimLeft(s[i:], " "), nil
}

func parsePathValue(s string) (value, rest string, err error) {
	if s[0] == '"' || s[0] == '\'' {
		return parsePathQuotedValue(s)
	}
	return parsePathUnquotedValue(s)
}

func parsePathQuotedValue(s string) (value, rest string, err error) {
	quote := s[0]
	end := strings.IndexByte(s[1:], quote)
	if end < 0 {
		return "", "", &pathParseError{msg: "unclosed quoted value"}
	}
	return s[1 : 1+end], strings.TrimLeft(s[1+end+1:], " "), nil
}

func parsePathUnquotedValue(s string) (value, rest string, err error) {
	j := 0
	for j < len(s) && s[j] != '/' {
		if !unicode.IsLetter(rune(s[j])) && !unicode.IsDigit(rune(s[j])) && s[j] != '-' && s[j] != '_' {
			return "", "", &pathParseError{msg: "unquoted value may only contain letters, digits, hyphen, underscore"}
		}
		j++
	}
	return s[:j], strings.TrimLeft(s[j:], " "), nil
}

// MatchMeta returns true if the given meta map satisfies the optional Period, Workstream, Owner filters.
// Empty filter field means "no constraint".
func (f *Filter) MatchMeta(meta map[string]interface{}) bool {
	if f == nil {
		return true
	}
	if f.Period != "" && metaStr(meta, "period") != f.Period {
		return false
	}
	if f.Workstream != "" && metaStr(meta, "workstream") != f.Workstream {
		return false
	}
	if f.Owner != "" && metaStr(meta, "owner") != f.Owner {
		return false
	}
	return true
}

func metaStr(meta map[string]interface{}, key string) string {
	if meta == nil {
		return ""
	}
	v, ok := meta[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// BuildPath returns the path-from-root for an entry: segments matching filterPath keys in order along the chain.
func BuildPath(ancestors []map[string]interface{}, entryMeta map[string]interface{}, filterPath []PathSegment) []PathSegment {
	if len(filterPath) == 0 {
		return nil
	}
	var path []PathSegment
	chain := append(append([]map[string]interface{}(nil), ancestors...), entryMeta)
	segIdx := 0
	for _, meta := range chain {
		if meta == nil || segIdx >= len(filterPath) {
			continue
		}
		want := filterPath[segIdx]
		if v := metaStr(meta, want.Key); v == want.Value {
			path = append(path, want)
			segIdx++
		}
	}
	return path
}

// PathMatches returns true if entryPath matches filterPath (prefix or full).
func PathMatches(entryPath, filterPath []PathSegment) bool {
	if len(filterPath) == 0 {
		return true
	}
	if len(entryPath) < len(filterPath) {
		return false
	}
	for i := range filterPath {
		if entryPath[i].Key != filterPath[i].Key || entryPath[i].Value != filterPath[i].Value {
			return false
		}
	}
	return true
}

// SelectEntries returns entries from f that match the filter (meta + path).
func SelectEntries(f *File, flt *Filter) []Entry {
	if flt == nil {
		return f.Roadmap
	}
	var out []Entry
	walkSelect(f.Roadmap, nil, flt, &out)
	return out
}

func walkSelect(entries []Entry, ancestors []map[string]interface{}, flt *Filter, out *[]Entry) {
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
		if pathOK && (metaOK || (flt.Period == "" && flt.Workstream == "" && flt.Owner == "")) {
			*out = append(*out, *e)
		}
		ancestorsNext := append(append([]map[string]interface{}(nil), ancestors...), meta)
		if len(e.Items) > 0 {
			walkSelect(e.Items, ancestorsNext, flt, out)
		}
	}
}

// mergeMeta returns meta merged from ancestors (child wins).
func mergeMeta(ancestors []map[string]interface{}, entryMeta map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	for _, m := range ancestors {
		for k, v := range m {
			out[k] = v
		}
	}
	for k, v := range entryMeta {
		out[k] = v
	}
	return out
}
