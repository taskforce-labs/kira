---
id: 022
title: title delineator undetected
status: done
kind: issue
assigned:
estimate: 0
created: 2026-01-24
tags: []
---

# title delineator undetected

The title delineator (colon `:`) is not being detected when creating work items directly with `kira new`. The command should parse a colon-separated title/description string, but currently treats the entire string as the title.

## Problem

When running:
```bash
kira new task todo "my title: my description"
```

**Current behavior:**
- The entire string `"my title: my description"` is treated as the title
- The description field remains empty
- The colon is included in the title

**Expected behavior:**
- Title should be: `"my title"`
- Description should be: `"my description"`
- The colon should be used as a delimiter to split title and description

## Root Cause

The issue occurs in `internal/commands/new.go` in the `parseWorkItemArgs` function. When parsing command-line arguments:

1. The function assigns arguments to `result.title` and `result.description` based on position
2. It does not check if the title argument contains a colon delimiter
3. There is existing logic for colon-based parsing in `parseIdeaTitleDescription()` (used when converting ideas to work items), but this logic is not applied to direct command-line arguments

**Code location:** `internal/commands/new.go:130-173` in `parseWorkItemArgs()`

The parsing logic currently works as:
- `args[0]` → template
- `args[1]` → status (if valid) or title
- `args[2]` → title or status (depending on previous)
- `args[3]` → description

But it never checks if `args[2]` (or wherever title ends up) contains a colon to split it.

## Technical Details

### Existing Colon Parsing Logic

There is already a function `parseIdeaTitleDescription()` in `internal/commands/ideas.go:380-421` that handles colon-based parsing:

```go
func parseIdeaTitleDescription(ideaText string) IdeaTitleDescription {
    // Check if idea contains a colon
    colonIndex := strings.Index(ideaText, ":")
    if colonIndex != -1 {
        // Has colon: text before colon = title, after = description
        result.Title = strings.TrimSpace(ideaText[:colonIndex])
        result.Description = strings.TrimSpace(ideaText[colonIndex+1:])
    }
    // ... fallback logic for no colon
}
```

This function is used when converting ideas to work items (`convertIdeaToWorkItem()`), but not when parsing direct command-line arguments.

### Where the Fix Should Go

**Yes, we should leverage the existing `parseIdeaTitleDescription()` function!**

The fix should be applied in `parseWorkItemArgs()` after parsing the arguments. The approach:

1. After `parseWorkItemArgs()` determines the title argument (stored in `result.title`)
2. Check if `result.title` contains a colon
3. If it does, call the existing `parseIdeaTitleDescription(result.title)` function
4. Update `result.title` and `result.description` with the parsed values
5. This reuses all the existing edge case handling (whitespace trimming, multiple colons, empty parts, etc.)

**Why reuse `parseIdeaTitleDescription()`:**
- It already handles all the edge cases we need (whitespace, multiple colons, empty title/description)
- It's well-tested (see `ideas_test.go`)
- It maintains consistency with how ideas are converted to work items
- The word-based fallback (first 5 words) won't hurt - if there's no colon, it just uses the whole string as title, which is the current behavior anyway

**Implementation approach:**
```go
// In parseWorkItemArgs(), after setting result.title:
if result.title != "" && strings.Contains(result.title, ":") {
    parsed := parseIdeaTitleDescription(result.title)
    result.title = parsed.Title
    // Only set description if it wasn't already explicitly provided
    if result.description == "" {
        result.description = parsed.Description
    }
}
```

**Note:** If description is explicitly provided as a separate argument (e.g., `kira new task todo "title: desc" "explicit desc"`), the explicit description should take precedence over the colon-split description.

### Edge Cases to Consider

- What if the title contains multiple colons? (e.g., `"title: part1: part2"`)
  - Should split on first colon only (consistent with idea parsing)
- What if description is provided both ways? (e.g., `kira new task todo "title: desc" "another desc"`)
  - Should the colon-split description take precedence, or should explicit description arg override?
- What if title starts or ends with colon? (e.g., `": description only"` or `"title:"`)
  - Should handle gracefully (similar to idea parsing logic)

## Related Code

- `internal/commands/new.go:130-173` - `parseWorkItemArgs()` - **where fix should be applied**
- `internal/commands/ideas.go:380-421` - `parseIdeaTitleDescription()` - **function to reuse for colon parsing**
- `internal/commands/new.go:447-508` - `convertIdeaToWorkItem()` - **example of using `parseIdeaTitleDescription()`**
- `internal/commands/ideas_test.go:212-254` - **tests for `parseIdeaTitleDescription()`** - can reference these for expected behavior

## Acceptance Criteria

- [ ] Running `kira new task todo "my title: my description"` correctly splits title and description
- [ ] Title is set to `"my title"` (without colon)
- [ ] Description is set to `"my description"`
- [ ] Works with existing argument parsing (template, status, title, description)
- [ ] Handles edge cases (multiple colons, empty parts, etc.)
- [ ] Unit tests added for colon parsing in direct command arguments
- [ ] Integration tests verify the behavior end-to-end
