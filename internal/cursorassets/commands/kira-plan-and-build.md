Build the given work item with <work-item-id>



Work through each slice in the work item using `kira slice current <work-item-id>`:
- Ensure there are unit tests for any code changes for that slice
- Ensure that `kira checks -t commit` passes before committing the slice
- Generate the commit message using `kira slice commit generate <work-item-id> <slice-name>`
- Commit the changes using `kira slice commit <work-item-id> <commit-message>`
- Repeat for each slice in the work item
- If there are no open tasks in the work item, mark the work item as complete using `kira slice commit <work-item-id>`