#!/bin/bash

# Kira Integration Test Script
# This script tests the basic functionality of Kira

set -e

# Flags
KEEP=0
for arg in "$@"; do
  case "$arg" in
    -k|--keep)
      KEEP=1
      ;;
    -h|--help)
      echo "Usage: $0 [--keep|-k]"
      echo "  --keep, -k  Preserve the generated test directory (skip cleanup)"
      exit 0
      ;;
  esac
done

ROOT_DIR="$(pwd)"
KIRA_BIN="$ROOT_DIR/kira"

echo "🧪 Testing Kira CLI Tool"
echo "========================="

# Build the tool
echo "📦 Building kira..."
go build -o "$KIRA_BIN" cmd/kira/main.go
echo "✅ Build successful"

# Create test directory
BASE_DIR="e2e-test"
mkdir -p "$BASE_DIR"
TEST_DIR="$BASE_DIR/test-kira-$(date +%s)"
TEST_DIR_ABS="$ROOT_DIR/$TEST_DIR"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

echo "📁 Created test directory: $TEST_DIR"

# Test 1: Initialize workspace
echo ""
echo "🔧 Test 1: Initialize workspace"
"$KIRA_BIN" init
if [ -d ".work" ]; then
    echo "✅ Workspace initialized successfully"
else
    echo "❌ Workspace initialization failed"
    exit 1
fi

# Test 2: Check directory structure
echo ""
echo "📂 Test 2: Check directory structure"
REQUIRED_DIRS=("0_backlog" "1_todo" "2_doing" "3_review" "4_done" "z_archive" "templates")
for dir in "${REQUIRED_DIRS[@]}"; do
    if [ -d ".work/$dir" ]; then
        echo "✅ Directory .work/$dir exists"
    else
        echo "❌ Directory .work/$dir missing"
        exit 1
    fi
done

# Ensure .gitkeep files exist in each directory
echo ""
echo "📄 Test 2b: Check .gitkeep files"
for dir in "${REQUIRED_DIRS[@]}"; do
    if [ -f ".work/$dir/.gitkeep" ]; then
        echo "✅ .gitkeep exists in .work/$dir"
    else
        echo "❌ .gitkeep missing in .work/$dir"
        exit 1
    fi
done

# Test 3: Check required files
echo ""
echo "📄 Test 3: Check required files"
# .work/IDEAS.md should exist
if [ -f ".work/IDEAS.md" ]; then
    echo "✅ File .work/IDEAS.md exists"
else
    echo "❌ File .work/IDEAS.md missing"
    exit 1
fi
# kira.yml should exist at repo root (test dir)
if [ -f "kira.yml" ]; then
    echo "✅ File kira.yml exists at root"
else
    echo "❌ File kira.yml missing at root"
    exit 1
fi

# Test 4: Check templates
echo ""
echo "📝 Test 4: Check templates"
TEMPLATE_FILES=("template.prd.md" "template.issue.md" "template.spike.md" "template.task.md")
for template in "${TEMPLATE_FILES[@]}"; do
    if [ -f ".work/templates/$template" ]; then
        echo "✅ Template .work/templates/$template exists"
    else
        echo "❌ Template .work/templates/$template missing"
        exit 1
    fi
done

# Test 5: Add an idea
echo ""
echo "💡 Test 5: Add an idea"
"$KIRA_BIN" idea "Test idea for integration testing"
if grep -q "Test idea for integration testing" .work/IDEAS.md; then
    echo "✅ Idea added successfully"
else
    echo "❌ Idea addition failed"
    exit 1
fi

# Test 6: Create a work item via 'kira new' with explicit inputs
echo ""
echo "📋 Test 6: Create a work item via 'kira new' (with --input, default status)"
"$KIRA_BIN" new prd "Test Feature From Inputs" \
  --input assigned=qa@example.com \
  --input estimate=5 \
  --input due=2025-12-31 \
  --input tags="frontend,api" \
  --input criteria1="Login works" \
  --input criteria2="Logout works" \
  --input context="Context text" \
  --input requirements="Requirements text" \
  --input implementation="Implementation notes" \
  --input release_notes="Release notes here"

# Determine created file path dynamically (prefer backlog default, then todo)
WORK_ITEM_PATH=$(find .work/0_backlog -maxdepth 1 -type f -name "*.prd.md" | head -n 1)
if [ -z "$WORK_ITEM_PATH" ]; then
  WORK_ITEM_PATH=$(find .work/1_todo -maxdepth 1 -type f -name "*.prd.md" | head -n 1)
fi
if [ -n "$WORK_ITEM_PATH" ] && [ -f "$WORK_ITEM_PATH" ]; then
    echo "✅ Work item created successfully: $WORK_ITEM_PATH"
else
    echo "❌ Work item creation failed"
    exit 1
fi

# Validate template fields were filled
if grep -q "^title: Test Feature From Inputs$" "$WORK_ITEM_PATH" && \
   grep -q "^status: backlog$" "$WORK_ITEM_PATH" && \
   grep -q "^kind: prd$" "$WORK_ITEM_PATH" && \
   grep -q "^assigned: qa@example.com$" "$WORK_ITEM_PATH" && \
   grep -q "^estimate: 5$" "$WORK_ITEM_PATH" && \
   grep -q "^due: 2025-12-31$" "$WORK_ITEM_PATH" && \
   grep -q "^tags: frontend,api$" "$WORK_ITEM_PATH" && \
   grep -q "^# Test Feature From Inputs$" "$WORK_ITEM_PATH"; then
    echo "✅ Template fields filled correctly from inputs"
else
    echo "❌ Template fields not filled as expected"
    echo "----- File contents -----"
    cat "$WORK_ITEM_PATH"
    echo "-------------------------"
    exit 1
fi

# Test 7: Lint check
echo ""
echo "🔍 Test 7: Lint check"
if "$KIRA_BIN" lint; then
    echo "✅ Lint check passed"
else
    echo "❌ Lint check failed"
    exit 1
fi

# Test 8: Doctor check
echo ""
echo "🩺 Test 8: Doctor check"
if "$KIRA_BIN" doctor; then
    echo "✅ Doctor check passed"
else
    echo "❌ Doctor check failed"
    exit 1
fi

# Test 9: Move work item
echo ""
echo "🔄 Test 9: Move work item"
"$KIRA_BIN" move 001 doing
MOVED_PATH=".work/2_doing/$(basename "$WORK_ITEM_PATH")"
if [ -f "$MOVED_PATH" ] && [ ! -f "$WORK_ITEM_PATH" ]; then
    echo "✅ Work item moved successfully"
else
    echo "❌ Work item move failed"
    echo "Expected moved path: $MOVED_PATH"
    echo "Original path: $WORK_ITEM_PATH"
    exit 1
fi

# Test 10: Help commands
echo ""
echo "❓ Test 10: Help commands and init flags"
if "$KIRA_BIN" --help > /dev/null; then
    echo "✅ Main help works"
else
    echo "❌ Main help failed"
    exit 1
fi

if "$KIRA_BIN" new --help > /dev/null; then
    echo "✅ New command help works"
else
    echo "❌ New command help failed"
    exit 1
fi

# Test 11: init flags: fill-missing and force
echo ""
echo "🧪 Test 11: init --fill-missing and --force"
# Remove a folder and create sentinel
rm -rf .work/3_review
touch .work/1_todo/sentinel.txt
if "$KIRA_BIN" init --fill-missing; then
  if [ -d .work/3_review ] && [ -f .work/1_todo/sentinel.txt ]; then
    echo "✅ fill-missing restored folder without overwriting existing files"
  else
    echo "❌ fill-missing behavior incorrect"
    exit 1
  fi
else
  echo "❌ init --fill-missing failed"
  exit 1
fi

# Test 12: Release command
echo ""
echo "🧪 Test 12: release command"
# Create a done item with Release Notes and run release
cat > .work/4_done/001-done-feature.prd.md << 'EOF'
---
id: 001
title: Done Feature
status: done
kind: prd
created: 2024-01-01
---

# Done Feature

## Release Notes
This is a release note entry.
EOF

if "$KIRA_BIN" release; then
  if ls .work/z_archive/*/4_done/001-done-feature.prd.md > /dev/null 2>&1 && [ ! -f .work/4_done/001-done-feature.prd.md ]; then
    echo "✅ Release archived done items and removed originals"
  else
    echo "❌ Release did not archive as expected"
    exit 1
  fi
  if grep -q "This is a release note entry." RELEASES.md; then
    echo "✅ Release notes appended to RELEASES.md"
  else
    echo "❌ Release notes missing in RELEASES.md"
    exit 1
  fi
else
  echo "❌ kira release failed"
  exit 1
fi

# Independently set up a sentinel to validate --force behavior (do not rely on previous tests)
touch .work/1_todo/sentinel.txt
if "$KIRA_BIN" init --force; then
  if [ ! -f .work/1_todo/sentinel.txt ] && [ -f .work/3_review/.gitkeep ]; then
    echo "✅ force overwrote workspace and recreated structure"
  else
    echo "❌ force behavior incorrect"
    exit 1
  fi
else
  echo "❌ init --force failed"
  exit 1
fi

# Test 13: abandon command
echo ""
echo "🧪 Test 13: abandon command"
# Re-init clean workspace
"$KIRA_BIN" init --force > /dev/null
# Create todo item and abandon by id with reason
cat > .work/1_todo/001-todo-one.prd.md << 'EOF'
---
id: 001
title: Todo One
status: todo
kind: prd
created: 2024-01-01
---
EOF
"$KIRA_BIN" abandon 001 "No longer needed"
if ls .work/z_archive/*/1_todo/001-todo-one.prd.md > /dev/null 2>&1 && [ ! -f .work/1_todo/001-todo-one.prd.md ]; then
  echo "✅ Abandon by id archived and removed original"
  if grep -q "## Abandonment" .work/z_archive/*/1_todo/001-todo-one.prd.md && grep -q "No longer needed" .work/z_archive/*/1_todo/001-todo-one.prd.md; then
    echo "✅ Abandonment reason added"
  else
    echo "❌ Abandonment reason missing"
    exit 1
  fi
else
  echo "❌ Abandon by id failed"
  exit 1
fi

# Create subfolder item and abandon folder
mkdir -p .work/1_todo/sub
cat > .work/1_todo/sub/002-todo-two.prd.md << 'EOF'
---
id: 002
title: Todo Two
status: todo
kind: prd
created: 2024-01-01
---
EOF
"$KIRA_BIN" abandon todo sub
if ls .work/z_archive/*/sub/002-todo-two.prd.md > /dev/null 2>&1 && [ ! -f .work/1_todo/sub/002-todo-two.prd.md ]; then
  echo "✅ Abandon folder archived and removed originals"
else
  echo "❌ Abandon folder failed"
  exit 1
fi

# Test 14: save command happy path in git repo
echo ""
echo "🧪 Test 14: save command"
"$KIRA_BIN" init --force > /dev/null
git init > /dev/null 2>&1
git config user.email test@example.com
git config user.name "Test User"
git add .
git commit -m "init" > /dev/null 2>&1
cat > .work/1_todo/001-save-test.prd.md << 'EOF'
---
id: 001
title: Save Test
status: todo
kind: prd
created: 2024-01-01
---

# Save Test
EOF
if "$KIRA_BIN" save "Custom commit message"; then
  if grep -q "^updated:" .work/1_todo/001-save-test.prd.md; then
    echo "✅ Updated timestamp added"
  else
    echo "❌ Updated timestamp missing"
    exit 1
  fi
  if git log -1 --pretty=%B | grep -q "Custom commit message"; then
    echo "✅ Commit with custom message created"
  else
    echo "❌ Commit message mismatch"
    exit 1
  fi
else
  echo "❌ kira save failed"
  exit 1
fi

# Test 15: move command with --commit flag
echo ""
echo "🧪 Test 15: move command with --commit flag"
"$KIRA_BIN" init --force > /dev/null
git init > /dev/null 2>&1
git config user.email test@example.com
git config user.name "Test User"
git add .
git commit -m "init" > /dev/null 2>&1
# Create a work item in todo
cat > .work/1_todo/002-move-commit-test.prd.md << 'EOF'
---
id: 002
title: Move Commit Test
status: todo
kind: prd
created: 2024-01-01
---

# Move Commit Test
EOF
git add .work/1_todo/002-move-commit-test.prd.md
git commit -m "Add work item" > /dev/null 2>&1
# Move with --commit flag
if "$KIRA_BIN" move 002 doing -c; then
  # Verify file was moved
  if [ -f ".work/2_doing/002-move-commit-test.prd.md" ] && [ ! -f ".work/1_todo/002-move-commit-test.prd.md" ]; then
    echo "✅ Work item moved successfully"
  else
    echo "❌ Work item move failed"
    exit 1
  fi
  # Verify commit was created
  COMMIT_MSG=$(git log -1 --pretty=%B)
  if echo "$COMMIT_MSG" | grep -q "Move prd 002 to doing"; then
    echo "✅ Commit message subject correct"
  else
    echo "❌ Commit message subject incorrect: $COMMIT_MSG"
    exit 1
  fi
  if echo "$COMMIT_MSG" | grep -q "Move Commit Test (todo -> doing)"; then
    echo "✅ Commit message body correct"
  else
    echo "❌ Commit message body incorrect: $COMMIT_MSG"
    exit 1
  fi
  # Verify commit includes both deletion and addition
  COMMIT_FILES=$(git show --name-status --pretty=format: HEAD | grep -E "^(A|D|R)" | grep "002-move-commit-test")
  if echo "$COMMIT_FILES" | grep -q "\.work/1_todo/002-move-commit-test.prd.md" && \
     echo "$COMMIT_FILES" | grep -q "\.work/2_doing/002-move-commit-test.prd.md"; then
    echo "✅ Commit includes both deletion and addition"
  else
    echo "❌ Commit does not include both changes"
    echo "Commit files:"
    git show --name-status HEAD | grep "002-move-commit-test"
    exit 1
  fi
  # Verify status was updated in file
  if grep -q "^status: doing$" .work/2_doing/002-move-commit-test.prd.md; then
    echo "✅ Status updated in file"
  else
    echo "❌ Status not updated in file"
    exit 1
  fi
else
  echo "❌ kira move --commit failed"
  exit 1
fi

# Test 16: start command - git operations (Phase 2)
echo ""
echo "🧪 Test 16: start command - git operations"

# Reset to clean state for start tests
rm -rf .git > /dev/null 2>&1
git init > /dev/null 2>&1
"$KIRA_BIN" init --force > /dev/null
git checkout -b main > /dev/null 2>&1
git config user.email test@example.com
git config user.name "Test User"
git add .
git commit -m "init" > /dev/null 2>&1

# Create a work item in todo
cat > .work/1_todo/003-start-test.prd.md << 'EOF'
---
id: 003
title: Start Test Feature
status: todo
kind: prd
created: 2024-01-01
---

# Start Test Feature
EOF
git add .work/1_todo/003-start-test.prd.md
git commit -m "Add work item" > /dev/null 2>&1

# Test: Start command creates worktree and branch
START_OUTPUT=$("$KIRA_BIN" start 003 2>&1)
if echo "$START_OUTPUT" | grep -q "Successfully started work on 003"; then
  echo "✅ Start command creates worktree and branch correctly"
else
  echo "❌ Start command failed"
  echo "Output: $START_OUTPUT"
  exit 1
fi

# Verify worktree was created
WORKTREE_ROOT=$(dirname "$(pwd)")
WORKTREE_NAME=$(basename "$(pwd)")_worktrees
if [ -d "$WORKTREE_ROOT/$WORKTREE_NAME/003-start-test-feature" ]; then
  echo "✅ Worktree directory created"
else
  echo "❌ Worktree directory not found at $WORKTREE_ROOT/$WORKTREE_NAME/003-start-test-feature"
  exit 1
fi

# Verify branch was created
if git branch --list "003-start-test-feature" | grep -q "003-start-test-feature"; then
  echo "✅ Branch created"
else
  echo "❌ Branch not created"
  exit 1
fi

# Cleanup worktree for subsequent tests
git worktree remove "$WORKTREE_ROOT/$WORKTREE_NAME/003-start-test-feature" --force > /dev/null 2>&1
git branch -D 003-start-test-feature > /dev/null 2>&1

# Test: Dry-run mode shows preview
DRY_RUN_OUTPUT=$("$KIRA_BIN" start 003 --dry-run 2>&1)
if echo "$DRY_RUN_OUTPUT" | grep -q "\[DRY RUN\]" && \
   echo "$DRY_RUN_OUTPUT" | grep -q "ID: 003" && \
   echo "$DRY_RUN_OUTPUT" | grep -q "Title: Start Test Feature" && \
   echo "$DRY_RUN_OUTPUT" | grep -q "Branch Name:"; then
  echo "✅ Dry-run mode shows correct preview"
else
  echo "❌ Dry-run mode output incorrect"
  echo "Output: $DRY_RUN_OUTPUT"
  exit 1
fi

# Test 17: start command - error cases
echo ""
echo "🧪 Test 17: start command - error cases"

# Test: Invalid work item ID format
if "$KIRA_BIN" start "../invalid" 2>&1 | grep -qi "invalid work item ID"; then
  echo "✅ Invalid ID format rejected correctly"
else
  echo "❌ Invalid ID format should be rejected"
  exit 1
fi

# Test: Work item not found
if "$KIRA_BIN" start 999 2>&1 | grep -qi "not found"; then
  echo "✅ Non-existent work item rejected correctly"
else
  echo "❌ Non-existent work item should be rejected"
  exit 1
fi

# Test: Invalid status-action flag
if "$KIRA_BIN" start 003 --status-action invalid 2>&1 | grep -qi "invalid status_action"; then
  echo "✅ Invalid status-action flag rejected correctly"
else
  echo "❌ Invalid status-action flag should be rejected"
  exit 1
fi

# Test 18: start command - status check behavior
echo ""
echo "🧪 Test 18: start command - status check behavior"

# Create a work item already in doing status
cat > .work/2_doing/004-already-doing.prd.md << 'EOF'
---
id: 004
title: Already Doing Feature
status: doing
kind: prd
created: 2024-01-01
---

# Already Doing Feature
EOF
git add .work/2_doing/004-already-doing.prd.md
git commit -m "Add already doing work item" > /dev/null 2>&1

# Test: Work item already in target status (doing) without --skip-status-check
if "$KIRA_BIN" start 004 2>&1 | grep -qi "already in 'doing' status"; then
  echo "✅ Work item in target status blocked correctly"
else
  echo "❌ Work item in target status should be blocked"
  exit 1
fi

# Test: Work item already in target status WITH --skip-status-check succeeds
START_SKIP_OUTPUT=$("$KIRA_BIN" start 004 --skip-status-check 2>&1)
if echo "$START_SKIP_OUTPUT" | grep -q "Successfully started work on 004"; then
  echo "✅ Work item in target status allowed with --skip-status-check"
  # Cleanup
  WORKTREE_ROOT=$(dirname "$(pwd)")
  WORKTREE_NAME=$(basename "$(pwd)")_worktrees
  git worktree remove "$WORKTREE_ROOT/$WORKTREE_NAME/004-already-doing-feature" --force > /dev/null 2>&1
  git branch -D 004-already-doing-feature > /dev/null 2>&1
else
  echo "❌ Work item in target status should be allowed with --skip-status-check"
  echo "Output: $START_SKIP_OUTPUT"
  exit 1
fi

# Test: status-action=none skips status check
START_NONE_OUTPUT=$("$KIRA_BIN" start 004 --status-action none 2>&1)
if echo "$START_NONE_OUTPUT" | grep -q "Successfully started work on 004"; then
  echo "✅ status-action=none skips status check correctly"
  # Cleanup
  git worktree remove "$WORKTREE_ROOT/$WORKTREE_NAME/004-already-doing-feature" --force > /dev/null 2>&1
  git branch -D 004-already-doing-feature > /dev/null 2>&1
else
  echo "❌ status-action=none should skip status check"
  echo "Output: $START_NONE_OUTPUT"
  exit 1
fi

# Test 19: start command - help
echo ""
echo "🧪 Test 19: start command - help"
if "$KIRA_BIN" start --help 2>&1 | grep -q "Creates a git worktree"; then
  echo "✅ Start command help works"
else
  echo "❌ Start command help failed"
  exit 1
fi

# Test 20: start command - IDE flags
echo ""
echo "🧪 Test 20: start command - IDE flags"

# Create a fresh work item for IDE tests
cat > .work/0_backlog/005-ide-test.prd.md << 'EOF'
---
kind: prd
id: 005
title: IDE Test Feature
status: backlog
created: 2025-01-08
---
# IDE Test Feature
Work item for testing IDE integration.
EOF
git add .work/0_backlog/005-ide-test.prd.md
git commit -m "Add IDE test work item" > /dev/null 2>&1

# Test: --no-ide flag skips IDE silently (should still create worktree)
START_NO_IDE_OUTPUT=$("$KIRA_BIN" start 005 --no-ide 2>&1)
if echo "$START_NO_IDE_OUTPUT" | grep -q "Successfully started work on 005"; then
  # Verify no IDE-related messages appear (only "Info: No IDE configured" should appear without --no-ide)
  if echo "$START_NO_IDE_OUTPUT" | grep -qi "Opening IDE"; then
    echo "❌ --no-ide flag should not show 'Opening IDE' message"
    exit 1
  fi
  echo "✅ --no-ide flag skips IDE silently"
  # Cleanup
  git worktree remove "$WORKTREE_ROOT/$WORKTREE_NAME/005-ide-test-feature" --force > /dev/null 2>&1
  git branch -D 005-ide-test-feature > /dev/null 2>&1
else
  echo "❌ --no-ide flag should allow worktree creation"
  echo "Output: $START_NO_IDE_OUTPUT"
  exit 1
fi

# Test: --ide flag with nonexistent command shows warning but succeeds
START_IDE_OUTPUT=$("$KIRA_BIN" start 005 --ide nonexistent-test-ide-cmd --skip-status-check 2>&1)
if echo "$START_IDE_OUTPUT" | grep -q "Successfully started work on 005"; then
  if echo "$START_IDE_OUTPUT" | grep -qi "Warning.*not found"; then
    echo "✅ --ide flag with invalid command shows warning but succeeds"
  else
    echo "✅ --ide flag creates worktree (IDE warning may vary by system)"
  fi
  # Cleanup
  git worktree remove "$WORKTREE_ROOT/$WORKTREE_NAME/005-ide-test-feature" --force > /dev/null 2>&1
  git branch -D 005-ide-test-feature > /dev/null 2>&1
else
  echo "❌ --ide flag with invalid command should still create worktree"
  echo "Output: $START_IDE_OUTPUT"
  exit 1
fi

# Test: dry-run shows IDE info
DRY_RUN_IDE_OUTPUT=$("$KIRA_BIN" start 005 --dry-run --ide "test-ide" 2>&1)
if echo "$DRY_RUN_IDE_OUTPUT" | grep -qi "IDE"; then
  echo "✅ Dry-run shows IDE information"
else
  echo "❌ Dry-run should show IDE information"
  echo "Output: $DRY_RUN_IDE_OUTPUT"
  exit 1
fi

# Test 21: start command - setup commands
echo ""
echo "🧪 Test 21: start command - setup commands"

# Update kira.yml to include workspace.setup
cat > kira.yml << 'EOF'
version: "1.0"
templates:
  prd: templates/template.prd.md
  issue: templates/template.issue.md
  spike: templates/template.spike.md
  task: templates/template.task.md
status_folders:
  backlog: 0_backlog
  todo: 1_todo
  doing: 2_doing
  review: 3_review
  done: 4_done
  archived: z_archive
default_status: backlog
validation:
  required_fields:
    - id
    - title
    - status
    - kind
    - created
  id_format: "^\\d{3}$"
  status_values:
    - backlog
    - todo
    - doing
    - review
    - done
    - released
    - abandoned
    - archived
commit:
  default_message: Update work items
  move_subject_template: "Move {type} {id} to {target_status}"
  move_body_template: "{title} ({current_status} -> {target_status})"
release:
  releases_file: RELEASES.md
  archive_date_format: "2006-01-02"
start:
  status_action: none
workspace:
  setup: "echo 'E2E_SETUP_RAN' > /tmp/kira-e2e-setup-test.txt"
EOF

# Create a fresh work item for setup tests
cat > .work/0_backlog/006-setup-test.prd.md << 'EOF'
---
kind: prd
id: 006
title: Setup Test Feature
status: backlog
created: 2025-01-08
---
# Setup Test Feature
Work item for testing setup commands.
EOF
git add kira.yml .work/0_backlog/006-setup-test.prd.md
git commit -m "Add setup test work item and kira.yml with setup" > /dev/null 2>&1

# Test: setup command runs and creates file
rm -f /tmp/kira-e2e-setup-test.txt
START_SETUP_OUTPUT=$("$KIRA_BIN" start 006 --no-ide 2>&1)
if echo "$START_SETUP_OUTPUT" | grep -q "Successfully started work on 006"; then
  if [ -f /tmp/kira-e2e-setup-test.txt ]; then
    if grep -q "E2E_SETUP_RAN" /tmp/kira-e2e-setup-test.txt; then
      echo "✅ Setup command executed successfully"
    else
      echo "❌ Setup command ran but output is incorrect"
      exit 1
    fi
    rm -f /tmp/kira-e2e-setup-test.txt
  else
    echo "❌ Setup command did not run (marker file not created)"
    exit 1
  fi
  # Cleanup
  git worktree remove "$WORKTREE_ROOT/$WORKTREE_NAME/006-setup-test-feature" --force > /dev/null 2>&1
  git branch -D 006-setup-test-feature > /dev/null 2>&1
else
  echo "❌ Start command with setup failed"
  echo "Output: $START_SETUP_OUTPUT"
  exit 1
fi

# Test: dry-run shows setup info
DRY_RUN_SETUP_OUTPUT=$("$KIRA_BIN" start 006 --dry-run 2>&1)
if echo "$DRY_RUN_SETUP_OUTPUT" | grep -q "Setup:"; then
  if echo "$DRY_RUN_SETUP_OUTPUT" | grep -q "Main Project:"; then
    echo "✅ Dry-run shows setup information"
  else
    echo "❌ Dry-run should show Main Project setup"
    exit 1
  fi
else
  echo "❌ Dry-run should show Setup section"
  echo "Output: $DRY_RUN_SETUP_OUTPUT"
  exit 1
fi

# Cleanup kira.yml (restore without setup)
cat > kira.yml << 'EOF'
version: "1.0"
templates:
  prd: templates/template.prd.md
  issue: templates/template.issue.md
  spike: templates/template.spike.md
  task: templates/template.task.md
status_folders:
  backlog: 0_backlog
  todo: 1_todo
  doing: 2_doing
  review: 3_review
  done: 4_done
  archived: z_archive
default_status: backlog
validation:
  required_fields:
    - id
    - title
    - status
    - kind
    - created
  id_format: "^\\d{3}$"
  status_values:
    - backlog
    - todo
    - doing
    - review
    - done
    - released
    - abandoned
    - archived
commit:
  default_message: Update work items
  move_subject_template: "Move {type} {id} to {target_status}"
  move_body_template: "{title} ({current_status} -> {target_status})"
release:
  releases_file: RELEASES.md
  archive_date_format: "2006-01-02"
EOF

# Cleanup
echo ""
if [ "$KEEP" -eq 1 ] || [ "${KEEP_TEST_DIR:-0}" -ne 0 ]; then
  echo "ℹ️ Skipping cleanup; test directory preserved at: $TEST_DIR_ABS"
else
  echo "🧹 Cleaning up..."
  cd "$ROOT_DIR"
  rm -rf "$TEST_DIR"
  rm -f "$KIRA_BIN"
  echo "✅ Cleanup complete"
fi

echo ""
echo "🎉 All tests passed! Kira is working correctly."
echo ""
echo "📊 Test Summary:"
echo "  ✅ Workspace initialization"
echo "  ✅ Directory structure"
echo "  ✅ Required files"
echo "  ✅ Template system"
echo "  ✅ Idea capture"
echo "  ✅ Work item creation"
echo "  ✅ Lint validation"
echo "  ✅ Doctor check"
echo "  ✅ Work item movement"
echo "  ✅ Help system"
echo "  ✅ Move with commit flag"
echo "  ✅ Start command validation"
echo "  ✅ Start command error handling"
echo "  ✅ Start command status check"
echo "  ✅ Start command IDE flags"
echo "  ✅ Start command setup commands"
echo ""
echo "🚀 Kira is ready for use!"

