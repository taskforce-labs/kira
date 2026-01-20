# Ideas

This file is for capturing quick ideas and thoughts that don't fit into formal work items yet.

## How to use
- Add ideas with timestamps using `kira idea "your idea here"`
- Or manually add entries below

## List
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
11. [2025-11-29] Command to split a work item into smaller work items.
12. [2025-11-29] Command to bulk update work items that match a pattern.
13. [2025-11-29] A game where you get to shoot kaju with an army of robots from the cli
14. [2025-11-29] Use git email to automatically assign work items to the current user.
15. [2025-11-30] kaiju easter egg: a hidden command that will show an ascii art kaiju
16. [2025-12-05] launch agent: command that will a launch an agent to pick up a work item
17. [2025-12-08] mcp process: runs a local mcp server in a single process that knows about all the kira based projects
18. [2025-12-08] Learn cli from gui: whenever a user interacts with the gui show the cli commands along the bottom of the screen
19. [2025-12-08] extract relevant context: a way to help LLMs get the relevant context about architecture for polyrepos codebases or large repos - some of the structure in kira.yml could be used to help LLMs get the relevant context - combined with cli access to ai clis
20. [2025-12-08] kira env branching: when kira work creates worktrees and branches the env files a treated so that multiple instances of the project can run locally
21. [2025-12-14] kira review context: an ability to include context history into the reivew - helpful when working with agents
22. [2026-01-05] id prefix: allow ids to have a prefix possibly via a template to add flexability to how it's formatted
23. [2026-01-05] non-numeric ids: allow ids to be generated with a short unique hash
24. [2026-01-07] init defaults: update init defaults with start config and options
25. [2026-01-09] kira project plans: when a prd or other work item is created, the start command will only submit prs for the repos that were affected
