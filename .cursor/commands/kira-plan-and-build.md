Plan the included work item and then build using the following approach

For each **slice** (not each task):

1. **Get current slice/tasks:** `kira slice current <work-item-id> --output json` (or `kira slice task current <work-item-id>`).
2. **Implement** all tasks in that slice; add/update tests.
3. **Verify:** `kira check -t commit` before committing. If checks fail, fix and re-run; only commit when they pass.
4. **Commit (one commit per slice):**
   - The commit must include **both** the code changes for the slice **and** the work item with that slice’s tasks marked done.
   - To avoid separate "Toggle task" commits: **edit the work item** to check the boxes for the tasks you just completed (in the `## Slices` section), then run `kira slice lint <work-item-id>` and fix any errors.
   - Stage the code files and the work item file, then run:
     `kira slice commit generate <work-item-id> | git commit -F -`
   - Do **not** run `kira slice task toggle current` before this commit if it would create its own commit; either skip toggle and edit the work item by hand, or run toggle only when you intend to commit **only** the work item (e.g. no code this step).
5. Move to the next slice and repeat.

When all slices are done, run `kira check -t done`. Mark the work item complete when everything passes.