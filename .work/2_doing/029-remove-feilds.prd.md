---
id: 029
title: remove fields
status: doing
kind: prd
assigned:
estimate: 0
created: 2026-01-29
due: 2026-01-29
tags: []
---

# remove fields

Remove `due` and `estimate` from templates and defaults. Fix `created` so it is authored consistently (system-set, not presented as user input). Align divergent ways of authoring work-item front matter where practical.

## Context

- **due and estimate**: These optional fields appear in all work-item templates (prd, issue, spike, task) but add noise and are often left empty or defaulted to `0` / placeholder dates. The product decision is to remove them from the default experience; teams that need them can add them via `kira.yml` `fields:` and custom templates.
- **created**: It is required and hardcoded in the validator (`WorkItem.Created`, `HardcodedFields`, `RequiredFields`), and the **new** command already sets it to today (`inputs["created"] = time.Now().Format("2006-01-02")`). Templates still show it as a user-editable "Creation date" placeholder (`<!--input-datetime[yyyy-mm-dd]:created:"Creation date"-->`), which is misleading: with `kira new` the value is overwritten, and with `replaceRemainingInputs` it becomes today. So `created` is effectively system-set but the template suggests otherwise. We should treat it as system-set everywhere (no interactive prompt, consistent placeholder behaviour).
- **Field authoring**: There are multiple ways front matter gets authored; they can get out of sync and create confusion about the single source of truth.

## How work-item fields are authored today (investigation)

### 1. Hardcoded struct (validator)

- **Where**: `internal/validation/validator.go` — `WorkItem` struct with `ID`, `Title`, `Status`, `Kind`, `Created`; `config.HardcodedFields`; `config.Validation.RequiredFields` (default includes these).
- **Behaviour**: Parsed into dedicated struct fields (or `Fields` for inline YAML); validated by `validateHardcodedField`; written first in fixed order in `writeYAMLFrontMatter`. Cannot be configured in `kira.yml` `fields:`.
- **Used for**: id, title, status, kind, created.

### 2. Config-driven fields (kira.yml `fields:`)

- **Where**: `internal/config/config.go` — `Config.Fields` map and `FieldConfig` (type, required, default, format, etc.).
- **Behaviour**: Validated by `validateConfiguredFields`; defaults applied by `ApplyFieldDefaults` (validator/doctor) and `applyFieldDefaultsToInputs` (new command). Written from `workItem.Fields` in sorted order after hardcoded fields. Optional fields (e.g. due, estimate) can exist only in config and be added by doctor when missing.
- **Used for**: assigned, due, estimate, tags, or any custom field defined in config.

### 3. Template placeholders (markdown templates)

- **Where**: Root `templates/*.md`, and default content in `internal/templates/templates.go` (`getPRDTemplate()`, etc.).
- **Syntax**: `<!--input-type:name:"description"-->` or `<!--input-strings[opts]:name:"desc"-->`. Processed by `ProcessTemplate` (replacement from `inputs` map) and `replaceRemainingInputs` (defaults: empty string, 0, today for datetime, `[]` for strings).
- **Behaviour**: Whatever keys appear in the template get replaced when creating a work item; no direct link to `config.Fields`. So we can have keys in templates that are not in config (e.g. due, estimate in templates but not in default kira.yml), and vice versa.

### 4. System-injected at runtime

- **created**: Set in `new.go` to today before template processing; also defaulted by `replaceRemainingInputs` for datetime placeholders.
- **updated**: Injected by the save command after `created` when missing; not in templates or config.

### Divergence and alignment opportunities

| Concern | Current state | Opportunity |
|--------|----------------|--------------|
| **due / estimate** | In all default templates; often 0 or placeholder. | Remove from default templates and from any default `fields` in code/docs so the default experience has no due/estimate. Teams that want them add `fields:` and optionally custom templates. |
| **created** | Required and hardcoded; new command sets today; template still shows as "Creation date" input. | Treat as system-set: keep setting in `new`, keep placeholder in template only so it gets today when not provided; do not prompt for it in interactive (or document that it’s overwritten). Optionally use a single non-interactive placeholder convention. |
| **Single source of field list** | Templates and config can disagree (templates have due/estimate; config may not). | After this PRD: default templates only contain hardcoded + assigned/tags (or whatever we agree as default). Optional fields (due, estimate, etc.) are added only via config + doctor or custom templates. |
| **Order and consistency** | Hardcoded fields written first in fixed order; then `Fields` in sorted order. | No change needed for this PRD; just reduce template keys so they match the desired default set. |

## Requirements

1. **Remove `due` and `estimate` from default templates**
   - Remove from root `templates/template.prd.md`, `template.issue.md`, `template.spike.md`, `template.task.md`.
   - Remove from default template strings in `internal/templates/templates.go` (getPRDTemplate, getIssueTemplate, getSpikeTemplate, getTaskTemplate).
   - PRD template: remove `due` line; issue/spike/task: remove `estimate` line (PRD already has both due and estimate).

2. **Align `created` authoring**
   - Keep `created` as required and hardcoded (no config change).
   - Ensure `kira new` continues to set `created` to today.
   - In default templates: keep a single `created` placeholder so processed output gets a date (today when not provided). Prefer a convention that does not present it as a user prompt (e.g. same placeholder but document that it’s system-set, or replace with a comment that new/doctor set it). Do not add `created` to interactive prompts if it would imply user choice; if it already is in GetTemplateInputs, consider excluding it from interactive so it’s always set by new/replaceRemainingInputs only.

3. **Do not add `due` or `estimate` to default config**
   - Default `kira.yml` and `DefaultConfig` should not define `fields.due` or `fields.estimate`. Docs/samples can show how to add them if desired.

4. **Tests and init**
   - Update tests that rely on default templates containing `due` or `estimate` (e.g. validator tests, new tests, integration tests).
   - `kira init` writes default templates from internal/templates; once those are updated, new repos get the reduced set.

## Acceptance Criteria

- [ ] Default templates (root and internal) contain no `due` or `estimate` keys.
- [ ] `created` remains required; `kira new` sets it to today; default templates result in a valid `created` value (e.g. today) without presenting it as a user-editable "Creation date" in interactive flow where that would be misleading.
- [ ] Default config (and default kira.yml from init) does not define `due` or `estimate` in `fields`.
- [ ] All tests updated and passing; `make check` and e2e pass.
- [ ] Existing work items that already have `due` or `estimate` in front matter remain valid (validation does not require removing them unless strict mode and config disallow unknown keys).

## Implementation Notes

- **Templates**: Edit `templates/*.md` and the four `get*Template()` strings in `internal/templates/templates.go`. Remove the lines for `due` (PRD) and `estimate` (all four).
- **created**: If interactive prompts are built from `GetTemplateInputs`, consider filtering out `created` so it is never prompted and only set by `collectInputs` / `replaceRemainingInputs`. Alternatively keep current behaviour and add a short doc note that "Creation date" is set by the tool when using `kira new`.
- **Tests**: Grep for `due`, `estimate`, and `created` in test content; adjust minimal work item YAML and template expectations. Validator tests that expect `due` or `estimate` in default templates should be updated or made config/template-agnostic.
- **Backward compatibility**: Validation does not require `due` or `estimate`; existing files with these keys are still valid. Only the default template set is reduced.

## Slices / Commits

Each slice is a separate commit. After each commit, run `make check` to ensure tests and lint pass.

| Slice | Commit intent | Check |
|-------|----------------|-------|
| **0** | PRD: add this Slices/Commits section and implementation breakdown. | `make check` |
| **1** | Remove `due` and `estimate` from repo-shipped `templates/*.md`; change `created` placeholder description to "Created (auto-set)". | `make check` |
| **2** | Remove `due` and `estimate` from `internal/templates/templates.go` (getPRDTemplate, getIssueTemplate, getSpikeTemplate, getTaskTemplate); update `created` wording. Update `internal/templates/templates_test.go` if needed. | `make check` |
| **3** | Update README.md: remove `due`/`estimate` from default examples; keep them only as optional fields in Field Configuration; adjust Work Item Format sample. | `make check` |
| **4** | Update `kira_e2e_tests.sh`: Test 6 no longer requires `estimate:`/`due:` in created file; remove or adjust `--input estimate=...`/`--input due=...` to match default templates. | `make check` |
| **5** | Verify `created` is never user-prompted (already set before interactive in new.go); optionally adjust `showTemplateInputs` so `created` is labelled as auto-set. | `make check`; `bash kira_e2e_tests.sh` |

## Release Notes

- Default work item templates no longer include `due` or `estimate`. Use `kira.yml` `fields:` and custom templates if you need these fields.
- `created` is set automatically when creating work items with `kira new`; the default template placeholder behaviour is unchanged so existing workflows remain valid.
