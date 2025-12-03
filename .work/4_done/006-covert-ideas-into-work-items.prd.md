---
id: 006
title: convert ideas into work items
status: done
kind: prd
assigned:
estimate: 0
created: 2025-11-29
due: 2025-11-29
tags: []
---

# convert ideas into work items

## Context
A user can add ideas to the IDEAS.md file with the command `kira idea "your idea here"`. This allows for quick capture of ideas without the distraction of fitting them into a formal template.

When a user is ready to work on an idea they can convert it to a work item with the command
```bash
kira new prd todo idea <idea number>
```

## Requirements

### Command behavior
The command should build on the existing `kira new prd todo` command by adding the idea number to the work item title.
```bash
kira new prd todo idea 1
```

**Command syntax clarification**: The `idea` keyword should be recognized as a special argument that indicates we're converting an idea. The format is:
- `kira new <template> <status> idea <idea_number>`
- The `idea` keyword must appear before the idea number
- The idea number must be a positive integer

Then I should get a file created under the todo folder. The work item ID should use the next available ID from `GetNextID()` (not the idea number), and the idea title should be used for the work item title.
```bash
.work/1_todo/007-dark-mode.prd.md
```

**ID assignment**: The work item ID should be generated using the existing `GetNextID()` function to maintain sequential ID consistency across all work items. The idea number is only used to locate the idea in IDEAS.md, not as the work item ID.

### Updates to how IDEAS.md file is managed and the ideas command

The ideas command should be updated to list the ideas with numbers instead of bullet points.
```bash
kira idea "dark mode: allow the user to toggle between light and dark mode"
```
Then I should get the idea number appended to the idea.
```bash
1. [2025-11-29] dark mode: allow the user to toggle between light and dark mode
```

**Idea title/description parsing**:
- Any text before the first colon (`:`) should be considered the idea title
- Any text after the first colon should be considered the idea description
- If an idea doesn't contain a colon, the first 5 words should be used as the title and the whole text should be used as the description
- Leading and trailing whitespace should be trimmed from both title and description
- If the title is empty after parsing (e.g., `: description only`), the first 5 words of the description should be used as the title
- If the idea has fewer than 5 words and no colon, the entire text should be used as the title and description should be empty

**Examples**:
- `"dark mode: allow the user to toggle between light and dark mode"` → Title: `"dark mode"`, Description: `"allow the user to toggle between light and dark mode"`
- `"add user authentication requirements with OAuth support"` → Title: `"add user authentication requirements with"`, Description: `"add user authentication requirements with OAuth support"`
- `"fix login bug"` → Title: `"fix login bug"`, Description: `""` (empty, since fewer than 5 words)
- `"implement a new feature"` → Title: `"implement a new feature"`, Description: `""` (empty, since exactly 4 words)

```bash
kira idea list
```
Then I should get a list of ideas in the console with numbers instead of bullet points.
```bash
1. [2025-11-29] dark mode: allow the user to toggle between light and dark mode
2. [2025-11-29] add user auth requirements
```

**`kira idea list` behavior**:
- Should display all ideas found under the `## Ideas` header
- Should show ideas in the order they appear in IDEAS.md
- Should display the idea number, timestamp, and full idea text
- Should handle empty IDEAS.md gracefully (show no ideas or empty message)
- Should exit with code 0 on success


## Implementation Notes

### Error Handling

The command should be able to handle the case where the idea number is not found in the IDEAS.md file.
```bash
kira new prd todo idea 100
```
Then I should get an error message saying "Idea 100 not found".
```bash
Error: Idea 100 not found
```

**Additional error cases to handle**:
- **Invalid idea number**: If the idea number is not a positive integer (e.g., `0`, `-1`, `abc`), return error: `Error: Invalid idea number: <value>`
- **IDEAS.md missing**: If `.work/IDEAS.md` doesn't exist, return error: `Error: IDEAS.md not found. Run 'kira init' first`
- **IDEAS.md empty**: If IDEAS.md exists but has no ideas under `## Ideas`, return error: `Error: No ideas found in IDEAS.md`
- **Missing ## Ideas header**: If IDEAS.md exists but doesn't contain `## Ideas` header, return error: `Error: IDEAS.md is missing the '## Ideas' header`
- **Malformed IDEAS.md**: If IDEAS.md exists but cannot be parsed, return error: `Error: Failed to parse IDEAS.md: <reason>`

### Idea Parsing Rules

Only text under the `## Ideas` header should be considered ideas.

**Idea format specification**:
- Ideas must be numbered lines starting with `<number>.` followed by a space
- The format is: `<number>. [<timestamp>] <idea text>`
- Ideas can span multiple lines (continuation lines should be indented)
- Blank lines between ideas are allowed and should be preserved
- Comments or other markdown under `## Ideas` that don't match the numbered format should be ignored

### Migration Strategy

**Handling existing IDEAS.md files**:
- When `kira idea` adds a new idea, it should use numbered format (`1. [timestamp] idea`)
- When `kira idea list` displays ideas, it should parse and display numbered format ideas (`1. [timestamp] idea`)
- Only ideas in numbered format will be recognized and can be converted to work items
- Users with existing IDEAS.md files will need to manually convert their ideas to numbered format or use a migration tool (if one is created in the future)

### Work Item Creation Details

**When converting an idea to a work item**:
- Extract the idea title:
  - If the idea contains a colon, use text before the first colon as the title
  - If the idea doesn't contain a colon, use the first 5 words as the title (or all words if fewer than 5)
- Extract the idea description:
  - If the idea contains a colon, use text after the first colon as the description
  - If the idea doesn't contain a colon, use the whole text as the description
- Use the extracted title as the work item title
- Use the extracted description as the work item description field (if the template supports it)
- Generate the work item ID using `GetNextID()` (not the idea number)
- Create the work item file using the standard naming convention: `<id>-<kebab-case-title>.<template>.md`
- The work item should be created in the specified status folder

**Idea status after conversion**:
- The idea should be removed from IDEAS.md after successful conversion to a work item
- After removal, remaining ideas should be renumbered sequentially to maintain continuous numbering (1, 2, 3, ...)
- If conversion fails (e.g., work item creation fails), the idea should remain in IDEAS.md unchanged

### File Naming Edge Cases

**Title sanitization for filenames**:
- Special characters in idea titles should be sanitized using the existing `kebabCase()` function
- Very long titles should be truncated (if needed, specify max length - suggest 50-100 characters)
- Empty titles should use a default like "untitled-idea" or the idea number
- Filename collisions should be handled (append a counter if file already exists)

### Command Argument Parsing

**Integration with existing `kira new` command**:
- The `idea` keyword should be detected as a special argument
- When `idea` is found in the arguments, the next argument should be treated as the idea number
- The command should support: `kira new <template> <status> idea <number>`
- The command should also support: `kira new idea <number>` (with template/status prompting if needed)
- The `idea` keyword should not conflict with existing status names or template names


## Technical Notes

### Testing Requirements

**Unit tests should cover**:
- Adding a new idea with numbered format
- Listing ideas with numbered format
- Converting an idea to a work item (idea found)
- Converting an idea to a work item (idea not found)
- Removing idea from IDEAS.md after successful conversion
- Renumbering remaining ideas after an idea is removed
- Preserving idea numbering when conversion fails (idea should remain)
- Parsing idea title/description with colon
- Parsing idea without colon (first 5 words as title, rest as description)
- Parsing idea without colon with fewer than 5 words (entire text as title)
- Parsing idea without colon with exactly 5 words (all 5 words as title, empty description)
- Handling empty IDEAS.md
- Handling IDEAS.md without `## Ideas` header
- Handling invalid idea numbers (0, negative, non-numeric)
- Handling malformed IDEAS.md
- Handling ideas with multiple colons (first colon is delimiter)
- Handling very long idea titles
- Handling special characters in idea titles
- Handling empty idea text
- Handling whitespace-only ideas
- Renumbering when ideas are added (sequential numbering)
- Preserving existing ideas when adding new ones
- Reading numbered format ideas
- Renumbering ideas correctly when one is removed (e.g., removing idea 2 from [1, 2, 3] should result in [1, 2])
- Handling edge case: removing the last idea
- Handling edge case: removing the first idea

**End-to-end tests should cover**:
- Full workflow: `kira idea "test idea"` → `kira idea list` → `kira new prd todo idea 1` → verify idea removed from IDEAS.md
- Error cases: `kira new prd todo idea 999` (not found)
- Error cases: `kira new prd todo idea abc` (invalid number)
- Error cases: `kira new prd todo idea 0` (invalid number)
- Multiple ideas: Add several ideas, list them, convert one, verify it's removed and others renumbered
- Ideas with and without colons
- Ideas without colons that are longer than 5 words (title truncation)
- Ideas without colons that are shorter than 5 words (full text as title)
- Ideas with special characters in titles
- Renumbering: Convert idea 2 from [1, 2, 3], verify remaining ideas are [1, 2]
- Renumbering: Convert idea 1 from [1, 2, 3], verify remaining ideas are [1, 2]

### Code Organization

- Create a new function `parseIdeasFile()` to read and parse IDEAS.md
- Create a new function `getIdeaByNumber()` to retrieve a specific idea by number
- Create a new function `addIdeaWithNumber()` to add ideas with sequential numbering
- Create a new function `listIdeas()` to display ideas
- Create a new function `removeIdeaByNumber()` to remove an idea and renumber remaining ideas
- Create a new function `renumberIdeas()` to update idea numbers sequentially after removal
- Create a new function `convertIdeaToWorkItem()` to handle the conversion logic (should call `removeIdeaByNumber()` after successful work item creation)
- Update `parseWorkItemArgs()` to detect the `idea` keyword
- Update `ideaCmd` to support subcommands (`idea <description>` and `idea list`)

**Implementation order for conversion**:
1. Parse IDEAS.md and find the idea by number
2. Extract title and description from the idea
3. Create the work item file
4. If work item creation succeeds, remove the idea from IDEAS.md and renumber remaining ideas
5. If work item creation fails, leave IDEAS.md unchanged

### Security Considerations

- Validate file paths to ensure they stay within `.work/` directory
- Sanitize idea numbers to prevent path traversal attacks
- Use existing `safeReadFile()` function for reading IDEAS.md
- Validate idea number is within reasonable bounds (e.g., 1-999999) to prevent DoS


## Release Notes

### User-Facing Changes

- **Breaking change**: `kira idea` command now adds ideas with numbered format instead of bullet points
- **New command**: `kira idea list` displays all ideas with numbers
- **New feature**: `kira new <template> <status> idea <number>` converts an idea to a work item
- **Breaking change**: Only numbered format ideas (`1. [timestamp] idea`) will be recognized. Existing bullet-point ideas will need to be manually converted to numbered format

### Known Limitations

- Idea numbers are sequential and cannot be manually assigned
- If an idea is manually deleted from IDEAS.md, numbers will not be automatically renumbered (only automatic renumbering happens when converting ideas to work items)
- Very long idea titles (>100 chars) may be truncated in filenames

### Future Enhancements (Out of Scope)

- Command to manually remove/delete ideas from IDEAS.md (ideas are automatically removed when converted)
- Command to manually renumber ideas (automatic renumbering happens when converting ideas or adding new ideas)
- Support for idea tags/categories
- Support for idea search/filtering
- Migration command to convert existing IDEAS.md files to numbered format
