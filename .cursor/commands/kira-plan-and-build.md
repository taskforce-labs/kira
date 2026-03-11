Plan the included work item and then build using the following approach

For each **slice** (not each task):

1. **Get current slice/tasks:** `kira slice current` (no argument when work item is clear from context). To specify the work item, pass one argument: `current` or `<work-item-id>` — e.g. `kira slice current 047-foo`. Use `--output json` for machine-readable output (includes `slice_number`). You can refer to a slice by its 1-based number or name in other commands (e.g. `kira slice show 1`, `kira slice task add 2 "desc"`). Same for `kira slice task current`.
2. **Implement** all tasks in that slice; add/update tests.
3. **Verify:** `kira check -t commit` before committing. If checks fail, fix and re-run; only commit when they pass.
4. **Commit (one commit per slice):**
   - The commit must include **both** the code changes for the slice **and** the work item with that slice’s tasks marked done.
   - To avoid separate "Toggle task" commits: **edit the work item** to check the boxes for the tasks you just completed (in the `## Slices` section), then run `kira slice lint [current | <work-item-id>]` and fix any errors.
   - Stage the code files and the work item file, then run:
     `kira slice commit generate [current | <work-item-id>] | git commit -F -`
   - Do **not** run `kira slice task done current` before this commit if it would create its own commit; either skip and edit the work item by hand, or run `kira slice task done current --commit` when you intend to commit **only** the work item (e.g. no code this step). Use `kira slice task done current --next` to mark done and see the next task plus progress summary; use `--hide-summary` on slice commands to suppress the one-line summary.
5. Move to the next slice and repeat.

When all slices are done, run `kira check -t done`.

Do not mark the work item as done or in review leave in todo.