# Ideas

This file is for capturing quick ideas and thoughts that don't fit into formal work items yet.

## How to use
- Add ideas with timestamps using `kira idea "your idea here"`
- Or manually add entries below

## Ideas

1. [2025-10-11] Local webserver to give non-technical folk a GUI to visualize the state of the system and make changes.

2. [2025-10-11] Kira should be able to create a static read only version of the GUI that can be hosted on github pages, s3, gitlab pages, etc.

3. [2025-10-11] Kira should have commands that make it easy to include as git submodules in other projects.

4. [2025-10-11] Kira should be able to organize wiki pages in a similar way to work items.

5. [2025-10-11] Kira init should offer to add a reference to the AGENTS.md file so agents know how to use the tool. Kira agent-reference command could be used to generate the reference also. The reference should be a markdown file called KIRA.md

6. [2025-10-11] kira.yml should sit outside of the .work directory and be a single file that contains all the configuration for the tool. If kira is managing docs we would want that in a separate folder called .docs.

7. [2025-10-11] Kira should be able to generate a list of all the work items in a given status.

8. [2025-10-11] Kira should be able to import data from Jira.

9. [2025-11-29] SQL lite DB to store work items and metadata for faster access of CLI commands.

10. [2025-11-29] Excel to extract reports for a query.

12. [2025-11-29] Command to split a work item into smaller work items.

13. [2025-11-29] Command to bulk update work items that match a pattern.

15. [2025-11-29] A game where you get to shoot kaju with an army of robots from the cli

16. [2025-11-29] Add a commit flag to the move command so that the work item is committed to git when moved.

18. [2025-11-29] List ideas with with numbers instead of bullet points.

19. [2025-11-29] Covert an idea to a work item with a command by entering the idea number and pressing enter.

21. [2025-11-29] Use git email to automatically assign work items to the current user.

22. [2025-11-29] List email addresses of all users from the git history and associate them a number that can be used to assign work items to the correct user. (these number will be in order of first commit to the repo so new users will have a higher number than existing users) kira config should allow for users to be ignored.
