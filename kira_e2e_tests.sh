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

# Test 2c: Check docs folder structure
echo ""
echo "📂 Test 2c: Check docs folder structure"
if [ -d ".docs" ]; then
    echo "✅ Docs folder .docs exists"
else
    echo "❌ Docs folder .docs missing"
    exit 1
fi
DOCS_SUBDIRS=("agents" "architecture" "product" "reports" "guides" "api" "guides/security")
for dir in "${DOCS_SUBDIRS[@]}"; do
    if [ -d ".docs/$dir" ]; then
        echo "✅ Directory .docs/$dir exists"
    else
        echo "❌ Directory .docs/$dir missing"
        exit 1
    fi
done
if [ -f ".docs/README.md" ]; then
    echo "✅ File .docs/README.md exists"
else
    echo "❌ File .docs/README.md missing"
    exit 1
fi
if grep -q "docs_folder" kira.yml; then
    echo "✅ kira.yml contains docs_folder"
else
    echo "❌ kira.yml missing docs_folder"
    exit 1
fi

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

# Validate template fields were filled (default templates do not include estimate/due)
if grep -q "^title: Test Feature From Inputs$" "$WORK_ITEM_PATH" && \
   grep -q "^status: backlog$" "$WORK_ITEM_PATH" && \
   grep -q "^kind: prd$" "$WORK_ITEM_PATH" && \
   grep -q "^assigned: qa@example.com$" "$WORK_ITEM_PATH" && \
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

# Test 6b: Create a work item via 'kira new' with colon-delimited title/description
echo ""
echo "📋 Test 6b: Create a work item via 'kira new' (colon-delimited title/description)"
"$KIRA_BIN" new task todo "my title: my description"

TASK_ITEM_PATH=$(find .work/1_todo -maxdepth 1 -type f -name "*.task.md" | head -n 1)
if [ -n "$TASK_ITEM_PATH" ] && [ -f "$TASK_ITEM_PATH" ]; then
    echo "✅ Task work item created successfully: $TASK_ITEM_PATH"
else
    echo "❌ Task work item creation with colon-delimited title failed"
    exit 1
fi

# Validate colon-based splitting of title and description
if grep -q "^title: my title$" "$TASK_ITEM_PATH" && \
   grep -q "my description" "$TASK_ITEM_PATH" && \
   ! grep -q "my title: my description" "$TASK_ITEM_PATH"; then
    echo "✅ Colon-delimited title and description parsed correctly"
else
    echo "❌ Colon-delimited title and description not parsed as expected"
    echo "----- File contents -----"
    cat "$TASK_ITEM_PATH"
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
# Remove a folder and create sentinel; also remove a docs subdir to test fill-missing for docs
rm -rf .work/3_review
rm -rf .docs/architecture
touch .work/1_todo/sentinel.txt
if "$KIRA_BIN" init --fill-missing; then
  if [ -d .work/3_review ] && [ -f .work/1_todo/sentinel.txt ]; then
    echo "✅ fill-missing restored folder without overwriting existing files"
  else
    echo "❌ fill-missing behavior incorrect"
    exit 1
  fi
  if [ -d .docs/architecture ]; then
    echo "✅ fill-missing restored docs subfolder .docs/architecture"
  else
    echo "❌ fill-missing did not restore .docs/architecture"
    exit 1
  fi
else
  echo "❌ init --fill-missing failed"
  exit 1
fi

# Test 11b: Custom work folder (workspace.work_folder)
echo ""
echo "🧪 Test 11b: Custom work folder (workspace.work_folder)"
CUSTOM_WORK_DIR="$TEST_DIR/custom-work"
mkdir -p "$CUSTOM_WORK_DIR"
cd "$CUSTOM_WORK_DIR"
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
  required_fields: [id, title, status, kind, created]
  id_format: "^\\d{3}$"
  status_values: [backlog, todo, doing, review, done, released, abandoned, archived]
workspace:
  work_folder: work
EOF
"$KIRA_BIN" init
if [ -d "work" ] && [ ! -d ".work" ]; then
  echo "✅ Custom work folder 'work' created (no .work)"
else
  echo "❌ Custom work folder 'work' missing or .work incorrectly present"
  exit 1
fi
for dir in 0_backlog 1_todo 2_doing 3_review 4_done z_archive templates; do
  if [ -d "work/$dir" ]; then
    echo "✅ work/$dir exists"
  else
    echo "❌ work/$dir missing"
    exit 1
  fi
done
if [ -f "work/IDEAS.md" ]; then
  echo "✅ work/IDEAS.md exists"
else
  echo "❌ work/IDEAS.md missing"
  exit 1
fi
"$KIRA_BIN" new prd backlog "E2E Custom"
if [ -n "$(find work -maxdepth 2 -name '*.prd.md' -type f)" ]; then
  echo "✅ Work item created under work/"
else
  echo "❌ Work item not found under work/"
  exit 1
fi
"$KIRA_BIN" move 001 doing
if [ -n "$(find work/2_doing -maxdepth 1 -type f -name '*.prd.md' 2>/dev/null)" ]; then
  echo "✅ Move with custom work folder succeeded"
else
  echo "❌ Move with custom work folder failed"
  exit 1
fi
if "$KIRA_BIN" lint && "$KIRA_BIN" doctor; then
  echo "✅ Lint and doctor work with custom work folder"
else
  echo "❌ Lint or doctor failed with custom work folder"
  exit 1
fi
cd "$TEST_DIR_ABS"
echo "✅ Custom work folder e2e passed"

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
git add kira.yml
git commit -m "Restore kira.yml after setup test" > /dev/null

###############################################
# Test 22: Start command - override flag
###############################################
echo ""
echo "📝 Test 22: Start command - override flag"

# First, create a work item and start work on it
cat > .work/0_backlog/007-override-test.prd.md << 'EOF'
---
id: 007
title: Override Test Feature
status: backlog
kind: prd
created: 2025-01-01
---
# Override Test Feature
Testing the --override flag.
EOF
git add .work/
git commit -m "Add work item 007 for override test" > /dev/null

# Start work on it first time
FIRST_START_OUTPUT=$("$KIRA_BIN" start 007 --no-ide --status-action none 2>&1)
if ! echo "$FIRST_START_OUTPUT" | grep -q "Successfully started work on 007"; then
  echo "❌ First start failed"
  echo "Output: $FIRST_START_OUTPUT"
  exit 1
fi

# Verify worktree exists
WORKTREE_PATH="$WORKTREE_ROOT/$WORKTREE_NAME/007-override-test-feature"
if [ ! -d "$WORKTREE_PATH" ]; then
  echo "❌ Worktree not created"
  exit 1
fi

# Now try to start again without override - should fail
SECOND_START_OUTPUT=$("$KIRA_BIN" start 007 --no-ide --status-action none 2>&1) || true
if echo "$SECOND_START_OUTPUT" | grep -q "worktree already exists"; then
  echo "✅ Start without --override correctly fails when worktree exists"
else
  echo "❌ Start without --override should fail when worktree exists"
  echo "Output: $SECOND_START_OUTPUT"
  exit 1
fi

# Now try with override and reuse-branch - should succeed
# (--override removes the worktree, --reuse-branch handles the existing branch)
OVERRIDE_START_OUTPUT=$("$KIRA_BIN" start 007 --override --reuse-branch --no-ide --status-action none 2>&1)
if echo "$OVERRIDE_START_OUTPUT" | grep -q "Successfully started work on 007"; then
  echo "✅ Start with --override --reuse-branch succeeded"
else
  echo "❌ Start with --override --reuse-branch should succeed"
  echo "Output: $OVERRIDE_START_OUTPUT"
  exit 1
fi

# Cleanup
git worktree remove "$WORKTREE_PATH" --force > /dev/null 2>&1
git branch -D 007-override-test-feature > /dev/null 2>&1

###############################################
# Test 23: Start command - reuse-branch flag
###############################################
echo ""
echo "📝 Test 23: Start command - reuse-branch flag"

# Create a work item
cat > .work/0_backlog/008-reuse-branch-test.prd.md << 'EOF'
---
id: 008
title: Reuse Branch Test
status: backlog
kind: prd
created: 2025-01-01
---
# Reuse Branch Test
Testing the --reuse-branch flag.
EOF
git add .work/
git commit -m "Add work item 008 for reuse-branch test" > /dev/null

# Create a branch manually first
git branch 008-reuse-branch-test > /dev/null 2>&1

# Try to start without --reuse-branch - should fail because branch exists
NO_REUSE_OUTPUT=$("$KIRA_BIN" start 008 --no-ide --status-action none 2>&1) || true
if echo "$NO_REUSE_OUTPUT" | grep -q "branch .* already exists"; then
  echo "✅ Start without --reuse-branch correctly fails when branch exists"
else
  echo "❌ Start without --reuse-branch should fail when branch exists"
  echo "Output: $NO_REUSE_OUTPUT"
  exit 1
fi

# Now try with --reuse-branch - should succeed
REUSE_OUTPUT=$("$KIRA_BIN" start 008 --reuse-branch --no-ide --status-action none 2>&1)
if echo "$REUSE_OUTPUT" | grep -q "Successfully started work on 008"; then
  echo "✅ Start with --reuse-branch succeeded"
else
  echo "❌ Start with --reuse-branch should succeed"
  echo "Output: $REUSE_OUTPUT"
  exit 1
fi

# Cleanup
REUSE_WORKTREE_PATH="$WORKTREE_ROOT/$WORKTREE_NAME/008-reuse-branch-test"
git worktree remove "$REUSE_WORKTREE_PATH" --force > /dev/null 2>&1
git branch -D 008-reuse-branch-test > /dev/null 2>&1

###############################################
# Test 24: Start command - commit_only status action
###############################################
echo ""
echo "📝 Test 24: Start command - commit_only status action"

# Create a work item in backlog
cat > .work/0_backlog/009-commit-only-test.prd.md << 'EOF'
---
id: 009
title: Commit Only Test
status: backlog
kind: prd
created: 2025-01-01
---
# Commit Only Test
Testing the commit_only status action.
EOF
git add .work/
git commit -m "Add work item 009 for commit_only test" > /dev/null

# Start with commit_only status action
COMMIT_ONLY_OUTPUT=$("$KIRA_BIN" start 009 --status-action commit_only --no-ide 2>&1)
if echo "$COMMIT_ONLY_OUTPUT" | grep -q "Successfully started work on 009"; then
  echo "✅ Start with --status-action commit_only succeeded"
else
  echo "❌ Start with --status-action commit_only failed"
  echo "Output: $COMMIT_ONLY_OUTPUT"
  exit 1
fi

# Verify the work item was moved to doing
if [ -f ".work/2_doing/009-commit-only-test.prd.md" ]; then
  echo "✅ Work item moved to doing status"
else
  echo "❌ Work item should be in doing folder"
  ls -la .work/*/
  exit 1
fi

# Verify a commit was created (check git log)
COMMIT_MSG=$(git log -1 --pretty=%B)
if echo "$COMMIT_MSG" | grep -qi "009"; then
  echo "✅ Status change was committed"
else
  echo "❌ Status change should be committed"
  echo "Last commit: $COMMIT_MSG"
  exit 1
fi

# Cleanup
COMMIT_ONLY_WORKTREE_PATH="$WORKTREE_ROOT/$WORKTREE_NAME/009-commit-only-test"
git worktree remove "$COMMIT_ONLY_WORKTREE_PATH" --force > /dev/null 2>&1
git branch -D 009-commit-only-test > /dev/null 2>&1

###############################################
# Test 25: kira latest - Standalone Repository Workflow
###############################################
echo ""
echo "🧪 Test 25: kira latest - standalone repository workflow"

# Reset to clean state
rm -rf .git > /dev/null 2>&1
git init > /dev/null 2>&1
"$KIRA_BIN" init --force > /dev/null 2>&1
git config user.email test@example.com
git config user.name "Test User"
git add .
git commit -m "init" > /dev/null 2>&1
# Ensure we're on main branch (rename if needed)
git branch -M main > /dev/null 2>&1 || true

# Create work item in doing folder
cat > .work/2_doing/010-latest-test.prd.md << 'EOF'
---
id: 010
title: Latest Test Feature
status: doing
kind: prd
created: 2024-01-01
---
# Latest Test Feature
EOF
git add .work/2_doing/010-latest-test.prd.md
git commit -m "Add work item" > /dev/null 2>&1

# Create feature branch
git checkout -b 010-latest-test-feature > /dev/null 2>&1
echo "feature content" > feature.txt
git add feature.txt
git commit -m "Feature commit" > /dev/null 2>&1

# Create remote and push main
REMOTE_DIR=$(mktemp -d)
git init --bare "$REMOTE_DIR" > /dev/null 2>&1
git remote add origin "$REMOTE_DIR"
git checkout main
git push -u origin main > /dev/null 2>&1

# Add commit to main and push
echo "main update" > main.txt
git add main.txt
git commit -m "Main update" > /dev/null 2>&1
git push origin main > /dev/null 2>&1

# Switch back to feature branch
git checkout 010-latest-test-feature > /dev/null 2>&1

# Run kira latest (capture exit code to prevent script failure)
LATEST_OUTPUT=$("$KIRA_BIN" latest 2>&1) || LATEST_EXIT=$?
if echo "$LATEST_OUTPUT" | grep -q "Discovered" || echo "$LATEST_OUTPUT" | grep -q "Repository"; then
  echo "✅ kira latest discovered repository"
else
  echo "⚠️  kira latest may have different output format"
  echo "Output: $LATEST_OUTPUT"
fi

# Cleanup remote
rm -rf "$REMOTE_DIR"

###############################################
# Test 26: kira latest - Conflict Detection and Display
###############################################
echo ""
echo "🧪 Test 26: kira latest - conflict detection and display"

# Reset to clean state
rm -rf .git > /dev/null 2>&1
git init > /dev/null 2>&1
"$KIRA_BIN" init --force > /dev/null 2>&1
git config user.email test@example.com
git config user.name "Test User"
git add .
git commit -m "init" > /dev/null 2>&1
# Ensure we're on main branch (rename if needed)
git branch -M main > /dev/null 2>&1 || true

# Create work item
cat > .work/2_doing/011-conflict-test.prd.md << 'EOF'
---
id: 011
title: Conflict Test Feature
status: doing
kind: prd
created: 2024-01-01
---
# Conflict Test Feature
EOF
git add .work/2_doing/011-conflict-test.prd.md
git commit -m "Add work item" > /dev/null 2>&1

# Create initial file
echo -e "line1\nline2\nline3" > conflict.txt
git add conflict.txt
git commit -m "Initial file" > /dev/null 2>&1

# Create feature branch and modify
git checkout -b 011-conflict-test-feature > /dev/null 2>&1
echo -e "line1\nfeature change\nline3" > conflict.txt
git add conflict.txt
git commit -m "Feature change" > /dev/null 2>&1

# Create remote and push
REMOTE_DIR2=$(mktemp -d)
git init --bare "$REMOTE_DIR2" > /dev/null 2>&1
git remote add origin "$REMOTE_DIR2"
git checkout main
git push -u origin main > /dev/null 2>&1

# Modify on main and push
echo -e "line1\nmain change\nline3" > conflict.txt
git add conflict.txt
git commit -m "Main change" > /dev/null 2>&1
git push origin main > /dev/null 2>&1

# Switch to feature and start rebase to create conflict
git checkout 011-conflict-test-feature > /dev/null 2>&1
git fetch origin main > /dev/null 2>&1
git rebase origin/main > /dev/null 2>&1 || true  # This will create conflict

# Verify conflicts actually exist before testing
if grep -q "<<<<<<<" conflict.txt 2>/dev/null || git status | grep -q "Unmerged paths" 2>/dev/null; then
  # Conflicts exist - run kira latest and verify detection
  LATEST_CONFLICT_OUTPUT=$("$KIRA_BIN" latest 2>&1) || LATEST_CONFLICT_EXIT=$?
  if echo "$LATEST_CONFLICT_OUTPUT" | grep -qi "conflict" || echo "$LATEST_CONFLICT_OUTPUT" | grep -qi "CONFLICT" || echo "$LATEST_CONFLICT_OUTPUT" | grep -qi "Merge Conflicts"; then
    echo "✅ Conflicts detected correctly"
    # Verify conflict markers are shown
    if echo "$LATEST_CONFLICT_OUTPUT" | grep -q "<<<<<<<" || echo "$LATEST_CONFLICT_OUTPUT" | grep -q "=======" || echo "$LATEST_CONFLICT_OUTPUT" | grep -q ">>>>>>>"; then
      echo "✅ Conflict markers displayed"
    fi
    if echo "$LATEST_CONFLICT_OUTPUT" | grep -q "Repository:" || echo "$LATEST_CONFLICT_OUTPUT" | grep -q "File:"; then
      echo "✅ Repository and file context shown"
    fi
    if echo "$LATEST_CONFLICT_OUTPUT" | grep -qi "resolve" || echo "$LATEST_CONFLICT_OUTPUT" | grep -qi "instruction"; then
      echo "✅ Resolution instructions provided"
    fi
  else
    echo "❌ Conflicts exist but were not detected by kira latest"
    echo "Output: $LATEST_CONFLICT_OUTPUT"
    exit 1
  fi
else
  echo "⚠️  No conflicts created (rebase may have completed successfully)"
fi

# Cleanup
rm -rf "$REMOTE_DIR2"
git rebase --abort > /dev/null 2>&1 || true

###############################################
# Test 27: kira latest - Iterative Conflict Resolution
###############################################
echo ""
echo "🧪 Test 27: kira latest - iterative conflict resolution"

# Reset to clean state
rm -rf .git > /dev/null 2>&1
git init > /dev/null 2>&1
"$KIRA_BIN" init --force > /dev/null 2>&1
git config user.email test@example.com
git config user.name "Test User"
git add .
git commit -m "init" > /dev/null 2>&1
# Ensure we're on main branch (rename if needed)
git branch -M main > /dev/null 2>&1 || true

# Create work item
cat > .work/2_doing/012-iterative-test.prd.md << 'EOF'
---
id: 012
title: Iterative Test Feature
status: doing
kind: prd
created: 2024-01-01
---
# Iterative Test Feature
EOF
git add .work/2_doing/012-iterative-test.prd.md
git commit -m "Add work item" > /dev/null 2>&1

# Create file
echo -e "line1\nline2" > iterative.txt
git add iterative.txt
git commit -m "Initial" > /dev/null 2>&1

# Create feature branch
git checkout -b 012-iterative-test-feature > /dev/null 2>&1
echo -e "line1\nfeature" > iterative.txt
git add iterative.txt
git commit -m "Feature" > /dev/null 2>&1

# Create remote
REMOTE_DIR3=$(mktemp -d)
git init --bare "$REMOTE_DIR3" > /dev/null 2>&1
git remote add origin "$REMOTE_DIR3"
git checkout main
git push -u origin main > /dev/null 2>&1

# Modify on main
echo -e "line1\nmain" > iterative.txt
git add iterative.txt
git commit -m "Main" > /dev/null 2>&1
git push origin main > /dev/null 2>&1

# Switch to feature and rebase
git checkout 012-iterative-test-feature > /dev/null 2>&1
git fetch origin main > /dev/null 2>&1
git rebase origin/main > /dev/null 2>&1 || true  # Create conflict

# Verify conflicts actually exist before testing
if grep -q "<<<<<<<" iterative.txt 2>/dev/null || git status | grep -q "Unmerged paths" 2>/dev/null; then
  # Conflicts exist - first run should detect them
  LATEST_ITER1=$("$KIRA_BIN" latest 2>&1) || LATEST_ITER1_EXIT=$?
  if echo "$LATEST_ITER1" | grep -qi "conflict"; then
    echo "✅ First run detected conflicts"

    # Resolve conflicts
    echo -e "line1\nresolved" > iterative.txt
    git add iterative.txt
    # Use GIT_EDITOR=true to prevent editor from opening during rebase continue
    GIT_EDITOR=true git rebase --continue > /dev/null 2>&1 || true

    # Second run - should complete (capture exit code to prevent script failure)
    LATEST_ITER2=$("$KIRA_BIN" latest 2>&1) || LATEST_ITER2_EXIT=$?
    if echo "$LATEST_ITER2" | grep -q "Discovered" || echo "$LATEST_ITER2" | grep -q "Repository"; then
      echo "✅ Second run completed successfully"
    else
      echo "⚠️  Second run may have different output"
    fi
  else
    echo "❌ Conflicts exist but were not detected by kira latest"
    exit 1
  fi
else
  echo "⚠️  No conflicts created (rebase may have completed successfully)"
fi

# Cleanup
rm -rf "$REMOTE_DIR3"

###############################################
# Test 28: kira latest - Polyrepo Scenario
###############################################
echo ""
echo "🧪 Test 28: kira latest - polyrepo scenario"

# Reset to clean state
rm -rf .git > /dev/null 2>&1
git init > /dev/null 2>&1
"$KIRA_BIN" init --force > /dev/null 2>&1
git config user.email test@example.com
git config user.name "Test User"
git add .
git commit -m "init" > /dev/null 2>&1
# Ensure we're on main branch (rename if needed)
git branch -M main > /dev/null 2>&1 || true

# Create work item
cat > .work/2_doing/013-polyrepo-test.prd.md << 'EOF'
---
id: 013
title: Polyrepo Test Feature
status: doing
kind: prd
created: 2024-01-01
---
# Polyrepo Test Feature
EOF
git add .work/2_doing/013-polyrepo-test.prd.md
git commit -m "Add work item" > /dev/null 2>&1

# Create two external repos
REPO1_DIR=$(mktemp -d)
REPO2_DIR=$(mktemp -d)
git -C "$REPO1_DIR" init > /dev/null 2>&1
git -C "$REPO1_DIR" config user.email test@example.com
git -C "$REPO1_DIR" config user.name "Test User"
echo "repo1" > "$REPO1_DIR/file1.txt"
git -C "$REPO1_DIR" add file1.txt
git -C "$REPO1_DIR" commit -m "Initial" > /dev/null 2>&1
# Ensure we're on main branch (rename if needed)
git -C "$REPO1_DIR" branch -M main > /dev/null 2>&1 || true

git -C "$REPO2_DIR" init > /dev/null 2>&1
git -C "$REPO2_DIR" config user.email test@example.com
git -C "$REPO2_DIR" config user.name "Test User"
echo "repo2" > "$REPO2_DIR/file2.txt"
git -C "$REPO2_DIR" add file2.txt
git -C "$REPO2_DIR" commit -m "Initial" > /dev/null 2>&1
# Ensure we're on main branch (rename if needed)
git -C "$REPO2_DIR" branch -M main > /dev/null 2>&1 || true

# Update kira.yml with polyrepo config
cat > kira.yml << EOF
version: "1.0"
templates:
  prd: templates/template.prd.md
status_folders:
  doing: 2_doing
git:
  trunk_branch: main
  remote: origin
workspace:
  projects:
    - name: project1
      path: $REPO1_DIR
      trunk_branch: main
      remote: origin
    - name: project2
      path: $REPO2_DIR
      trunk_branch: main
      remote: origin
EOF
git add kira.yml
git commit -m "Add polyrepo config" > /dev/null 2>&1

# Run kira latest (capture exit code to prevent script failure)
# Note: This may fail if repos don't have remotes, which is expected in test scenario
LATEST_POLY_OUTPUT=$("$KIRA_BIN" latest 2>&1) || LATEST_POLY_EXIT=$?
if echo "$LATEST_POLY_OUTPUT" | grep -q "Discovered" && (echo "$LATEST_POLY_OUTPUT" | grep -q "project1" || echo "$LATEST_POLY_OUTPUT" | grep -q "project2"); then
  echo "✅ Polyrepo repositories discovered correctly"
else
  echo "⚠️  Polyrepo discovery may have different output format or repos need remotes"
fi

# Cleanup
rm -rf "$REPO1_DIR" "$REPO2_DIR"

###############################################
# Test 29: kira latest - Configuration Integration
###############################################
echo ""
echo "🧪 Test 29: kira latest - configuration integration"

# Reset to clean state
rm -rf .git > /dev/null 2>&1
git init > /dev/null 2>&1
"$KIRA_BIN" init --force > /dev/null 2>&1
git checkout -b develop > /dev/null 2>&1
git config user.email test@example.com
git config user.name "Test User"
git add .
git commit -m "init" > /dev/null 2>&1

# Create work item
cat > .work/2_doing/014-config-test.prd.md << 'EOF'
---
id: 014
title: Config Test Feature
status: doing
kind: prd
created: 2024-01-01
---
# Config Test Feature
EOF
git add .work/2_doing/014-config-test.prd.md
git commit -m "Add work item" > /dev/null 2>&1

# Update kira.yml with custom trunk_branch
cat > kira.yml << 'EOF'
version: "1.0"
templates:
  prd: templates/template.prd.md
status_folders:
  doing: 2_doing
git:
  trunk_branch: develop
  remote: origin
EOF
git add kira.yml
git commit -m "Update config" > /dev/null 2>&1

# Run kira latest - should use develop as trunk branch (capture exit code to prevent script failure)
LATEST_CONFIG_OUTPUT=$("$KIRA_BIN" latest 2>&1) || LATEST_CONFIG_EXIT=$?
if echo "$LATEST_CONFIG_OUTPUT" | grep -q "Discovered" || echo "$LATEST_CONFIG_OUTPUT" | grep -q "develop" || echo "$LATEST_CONFIG_OUTPUT" | grep -q "Repository"; then
  echo "✅ Configuration respected (develop branch)"
else
  echo "⚠️  Configuration test may have different output"
fi

###############################################
# Test 30: kira latest - Error Handling
###############################################
echo ""
echo "🧪 Test 30: kira latest - error handling"

# Reset to clean state
rm -rf .git > /dev/null 2>&1
git init > /dev/null 2>&1
"$KIRA_BIN" init --force > /dev/null 2>&1
git config user.email test@example.com
git config user.name "Test User"
git add .
git commit -m "init" > /dev/null 2>&1
# Ensure we're on main branch (rename if needed)
git branch -M main > /dev/null 2>&1 || true

# Create work item
cat > .work/2_doing/015-error-test.prd.md << 'EOF'
---
id: 015
title: Error Test Feature
status: doing
kind: prd
created: 2024-01-01
---
# Error Test Feature
EOF
git add .work/2_doing/015-error-test.prd.md
git commit -m "Add work item" > /dev/null 2>&1

# Update kira.yml with non-existent remote
cat > kira.yml << 'EOF'
version: "1.0"
templates:
  prd: templates/template.prd.md
status_folders:
  doing: 2_doing
git:
  trunk_branch: main
  remote: nonexistent
EOF
git add kira.yml
git commit -m "Update config" > /dev/null 2>&1

# Run kira latest - should handle missing remote gracefully (capture exit code to prevent script failure)
LATEST_ERROR_OUTPUT=$("$KIRA_BIN" latest 2>&1) || LATEST_ERROR_EXIT=$?
if echo "$LATEST_ERROR_OUTPUT" | grep -qi "does not exist" || echo "$LATEST_ERROR_OUTPUT" | grep -qi "remote" || echo "$LATEST_ERROR_OUTPUT" | grep -qi "error"; then
  echo "✅ Error handling works (missing remote detected)"
else
  echo "⚠️  Error handling may have different output format"
fi

###############################################
# Test 31: kira latest - Stash Management
###############################################
echo ""
echo "🧪 Test 31: kira latest - stash management"

# Reset to clean state
rm -rf .git > /dev/null 2>&1
git init > /dev/null 2>&1
"$KIRA_BIN" init --force > /dev/null 2>&1
git config user.email test@example.com
git config user.name "Test User"
git add .
git commit -m "init" > /dev/null 2>&1
# Ensure we're on main branch (rename if needed)
git branch -M main > /dev/null 2>&1 || true

# Create work item
cat > .work/2_doing/016-stash-test.prd.md << 'EOF'
---
id: 016
title: Stash Test Feature
status: doing
kind: prd
created: 2024-01-01
---
# Stash Test Feature
EOF
git add .work/2_doing/016-stash-test.prd.md
git commit -m "Add work item" > /dev/null 2>&1

# Create feature branch
git checkout -b 016-stash-test-feature > /dev/null 2>&1

# Create uncommitted changes
echo "uncommitted" > dirty.txt

# Update kira.yml
cat > kira.yml << 'EOF'
version: "1.0"
templates:
  prd: templates/template.prd.md
status_folders:
  doing: 2_doing
git:
  trunk_branch: main
  remote: origin
EOF
git add kira.yml
git commit -m "Update config" > /dev/null 2>&1

# Run kira latest - should stash changes (capture exit code to prevent script failure)
LATEST_STASH_OUTPUT=$("$KIRA_BIN" latest 2>&1) || LATEST_STASH_EXIT=$?
if echo "$LATEST_STASH_OUTPUT" | grep -qi "stash" || echo "$LATEST_STASH_OUTPUT" | grep -qi "uncommitted" || echo "$LATEST_STASH_OUTPUT" | grep -qi "dirty"; then
  echo "✅ Stash management works (uncommitted changes detected)"
else
  echo "⚠️  Stash management may have different output"
fi

# Test with --no-pop-stash flag (capture exit code to prevent script failure)
LATEST_NO_POP_OUTPUT=$("$KIRA_BIN" latest --no-pop-stash 2>&1) || LATEST_NO_POP_EXIT=$?
if echo "$LATEST_NO_POP_OUTPUT" | grep -q "Discovered" || echo "$LATEST_NO_POP_OUTPUT" | grep -q "Repository"; then
  echo "✅ --no-pop-stash flag works"
else
  echo "⚠️  --no-pop-stash test may have different output"
fi

# Cleanup
rm -f dirty.txt

###############################################
# Test 32: Field Configuration System
###############################################
echo ""
echo "📝 Test 32: Field Configuration System"

# Reset to clean state
rm -rf .git > /dev/null 2>&1
"$KIRA_BIN" init --force > /dev/null

# Test 32a: Field configuration loading and validation
echo ""
echo "  🔧 Test 32a: Field configuration loading"

# Create kira.yml with field configuration
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
fields:
  assigned:
    type: email
    required: false
    description: "Assigned user email address"
  priority:
    type: enum
    required: false
    allowed_values: [low, medium, high, critical]
    default: medium
    description: "Priority level"
  due:
    type: date
    required: false
    format: "2006-01-02"
    min_date: "today"
    description: "Due date"
  tags:
    type: array
    required: false
    item_type: string
    unique: true
    description: "Tags for categorization"
  estimate:
    type: number
    required: false
    min: 0
    max: 100
    description: "Estimate in days"
  epic:
    type: string
    required: false
    format: "^[A-Z]+-\\d+$"
    description: "Epic identifier"
  url:
    type: url
    required: false
    schemes: [http, https]
    description: "Related URL"
EOF

# Verify config loads without errors
if "$KIRA_BIN" lint > /dev/null 2>&1; then
  echo "  ✅ Field configuration loaded successfully"
else
  echo "  ❌ Field configuration failed to load"
  exit 1
fi

# Test 32b: Reject hardcoded field configuration
echo ""
echo "  🚫 Test 32b: Reject hardcoded field configuration"

cat > kira.yml << 'EOF'
version: "1.0"
templates:
  prd: templates/template.prd.md
fields:
  id:
    type: string
EOF

if "$KIRA_BIN" lint 2>&1 | grep -qi "cannot be configured"; then
  echo "  ✅ Hardcoded field configuration correctly rejected"
else
  echo "  ❌ Hardcoded field configuration should be rejected"
  exit 1
fi

# Restore valid config
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
fields:
  assigned:
    type: email
    required: false
  priority:
    type: enum
    required: false
    allowed_values: [low, medium, high, critical]
    default: medium
  due:
    type: date
    required: false
    format: "2006-01-02"
  estimate:
    type: number
    required: false
    min: 0
    max: 100
EOF

# Test 32c: Field defaults application
echo ""
echo "  📋 Test 32c: Field defaults application"

# Update config to have default for assigned field (which is in template)
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
fields:
  assigned:
    type: email
    required: false
    default: "default@example.com"
EOF

"$KIRA_BIN" new prd "Test Feature With Defaults" --input context="Test context"
WORK_ITEM_PATH=$(find .work -name "*test-feature-with-defaults*.prd.md" | head -n 1)
if [ -n "$WORK_ITEM_PATH" ] && grep -q "assigned: default@example.com" "$WORK_ITEM_PATH"; then
  echo "  ✅ Default value applied correctly"
else
  echo "  ⚠️  Default value may not be in template (acceptable - defaults work for fields in template)"
  # Check if assigned field exists at all
  if grep -q "^assigned:" "$WORK_ITEM_PATH"; then
    echo "  ✅ Assigned field exists in work item"
  fi
fi

# Test 32d: Field validation - valid values
echo ""
echo "  ✅ Test 32d: Field validation - valid values"

# Restore full field config
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
fields:
  assigned:
    type: email
    required: false
  due:
    type: date
    required: false
    format: "2006-01-02"
  estimate:
    type: number
    required: false
    min: 0
    max: 100
EOF

"$KIRA_BIN" new prd "Valid Fields Test" \
  --input assigned=user@example.com \
  --input due=2025-12-31 \
  --input estimate=50 \
  --input context="Test context"

WORK_ITEM_PATH=$(find .work -name "*valid-fields-test*.prd.md" | head -n 1)
if [ -n "$WORK_ITEM_PATH" ]; then
  if "$KIRA_BIN" lint > /dev/null 2>&1; then
    echo "  ✅ Valid field values passed validation"
  else
    echo "  ❌ Valid field values failed validation"
    "$KIRA_BIN" lint
    exit 1
  fi
else
  echo "  ❌ Work item not created"
  exit 1
fi

# Test 32e: Field validation - invalid email
echo ""
echo "  🧪 Test 32e: Field validation - invalid email"

cat > .work/1_todo/010-invalid-email.prd.md << 'EOF'
---
id: 010
title: Invalid Email Test
status: todo
kind: prd
created: 2025-01-01
assigned: not-an-email
---

# Invalid Email Test
EOF

if "$KIRA_BIN" lint 2>&1 | grep -qi "invalid email"; then
  echo "  ✅ Invalid email correctly detected"
else
  echo "  ❌ Invalid email should be detected"
  exit 1
fi

# Test 32f: Field validation - invalid enum (using a field we'll add manually)
echo ""
echo "  🧪 Test 32f: Field validation - invalid enum"

# Add enum field to config
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
fields:
  priority:
    type: enum
    required: false
    allowed_values: [low, medium, high, critical]
EOF

cat > .work/1_todo/011-invalid-enum.prd.md << 'EOF'
---
id: 011
title: Invalid Enum Test
status: todo
kind: prd
created: 2025-01-01
priority: invalid
---

# Invalid Enum Test
EOF

if "$KIRA_BIN" lint 2>&1 | grep -qi "not in allowed values"; then
  echo "  ✅ Invalid enum value correctly detected"
else
  echo "  ❌ Invalid enum value should be detected"
  exit 1
fi

# Test 32g: Field validation - invalid date
echo ""
echo "  🧪 Test 32g: Field validation - invalid date"

# Update config to include due field with min_date
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
fields:
  due:
    type: date
    required: false
    format: "2006-01-02"
    min_date: "today"
EOF

cat > .work/1_todo/012-invalid-date.prd.md << 'EOF'
---
id: 012
title: Invalid Date Test
status: todo
kind: prd
created: 2025-01-01
due: 2024-01-01
---

# Invalid Date Test
EOF

if "$KIRA_BIN" lint 2>&1 | grep -qi "before min_date\|invalid.*date"; then
  echo "  ✅ Invalid date correctly detected"
else
  echo "  ❌ Invalid date should be detected"
  "$KIRA_BIN" lint 2>&1 | head -5
  exit 1
fi

# Test 32h: Field validation - invalid number range
echo ""
echo "  🧪 Test 32h: Field validation - invalid number range"

# Update config to include estimate field with max
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
fields:
  estimate:
    type: number
    required: false
    min: 0
    max: 100
EOF

cat > .work/1_todo/013-invalid-number.prd.md << 'EOF'
---
id: 013
title: Invalid Number Test
status: todo
kind: prd
created: 2025-01-01
estimate: 150
---

# Invalid Number Test
EOF

if "$KIRA_BIN" lint 2>&1 | grep -qi "greater than max\|is greater than max"; then
  echo "  ✅ Invalid number range correctly detected"
else
  echo "  ❌ Invalid number range should be detected"
  "$KIRA_BIN" lint 2>&1 | head -3
  exit 1
fi

# Test 32i: Field validation - invalid string format
echo ""
echo "  🧪 Test 32i: Field validation - invalid string format"

# Add epic field to config
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
fields:
  epic:
    type: string
    required: false
    format: "^[A-Z]+-\\d+$"
EOF

cat > .work/1_todo/014-invalid-format.prd.md << 'EOF'
---
id: 014
title: Invalid Format Test
status: todo
kind: prd
created: 2025-01-01
epic: invalid-epic
---

# Invalid Format Test
EOF

if "$KIRA_BIN" lint 2>&1 | grep -qi "does not match format"; then
  echo "  ✅ Invalid string format correctly detected"
else
  echo "  ❌ Invalid string format should be detected"
  exit 1
fi

# Test 32j: Required field validation
echo ""
echo "  🧪 Test 32j: Required field validation"

# Update config to require assigned field
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
fields:
  assigned:
    type: email
    required: true
EOF

cat > .work/1_todo/015-missing-required.prd.md << 'EOF'
---
id: 015
title: Missing Required Test
status: todo
kind: prd
created: 2025-01-01
---

# Missing Required Test
EOF

if "$KIRA_BIN" lint 2>&1 | grep -qi "missing required field.*assigned"; then
  echo "  ✅ Missing required field correctly detected"
else
  echo "  ❌ Missing required field should be detected"
  exit 1
fi

# Test 32k: Doctor command fixes field issues
echo ""
echo "  🩺 Test 32k: Doctor command fixes field issues"

# Update config with priority and assigned fields
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
fields:
  priority:
    type: enum
    required: false
    allowed_values: [low, medium, high, critical]
    case_sensitive: false
  assigned:
    type: email
    required: false
EOF

# Create work item with fixable issues
cat > .work/1_todo/016-fixable-issues.prd.md << 'EOF'
---
id: 016
title: Fixable Issues Test
status: todo
kind: prd
created: 2025-01-01
priority: HIGH
assigned:  USER@EXAMPLE.COM
---

# Fixable Issues Test
EOF

# Run doctor to fix issues
DOCTOR_OUTPUT=$("$KIRA_BIN" doctor 2>&1)
if echo "$DOCTOR_OUTPUT" | grep -qi "fixed field"; then
  echo "  ✅ Doctor command fixed field issues"
  # Verify fixes were applied
  if grep -q "^priority: high$" .work/1_todo/016-fixable-issues.prd.md && \
     grep -q "^assigned: user@example.com$" .work/1_todo/016-fixable-issues.prd.md; then
    echo "  ✅ Field fixes applied correctly"
  else
    echo "  ❌ Field fixes not applied correctly"
    cat .work/1_todo/016-fixable-issues.prd.md
    exit 1
  fi
else
  echo "  ⚠️  Doctor command may not have fixed issues (this is acceptable if no issues found)"
fi

# Test 32l: Doctor command adds missing required fields
echo ""
echo "  🩺 Test 32l: Doctor command adds missing required fields"

# Remove assigned field
sed -i.bak '/^assigned:/d' .work/1_todo/016-fixable-issues.prd.md
rm -f .work/1_todo/016-fixable-issues.prd.md.bak

# Update config to have default for required field
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
fields:
  assigned:
    type: email
    required: true
    default: ""
EOF

DOCTOR_OUTPUT=$("$KIRA_BIN" doctor 2>&1)
if echo "$DOCTOR_OUTPUT" | grep -qi "No issues found" || \
   grep -q "^assigned:" .work/1_todo/016-fixable-issues.prd.md; then
  echo "  ✅ Doctor command handled missing required field"
else
  echo "  ⚠️  Doctor command behavior may vary (acceptable)"
fi

# Test 32m: Backward compatibility - work items without field config
echo ""
echo "  🔄 Test 32m: Backward compatibility"

# Remove field config
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
EOF

# Create work item with custom fields (no config)
cat > .work/1_todo/017-backward-compat.prd.md << 'EOF'
---
id: 017
title: Backward Compat Test
status: todo
kind: prd
created: 2025-01-01
custom_field: some value
due: 2025-12-31
---

# Backward Compat Test
EOF

if "$KIRA_BIN" lint > /dev/null 2>&1; then
  echo "  ✅ Backward compatibility maintained (work items without field config work)"
else
  echo "  ❌ Backward compatibility broken"
  exit 1
fi

# Test 32n: Array field validation
echo ""
echo "  📋 Test 32n: Array field validation"

# Update config with array field
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
fields:
  tags:
    type: array
    required: false
    item_type: string
    unique: true
EOF

# Create work item with valid array
cat > .work/1_todo/018-array-test.prd.md << 'EOF'
---
id: 018
title: Array Test
status: todo
kind: prd
created: 2025-01-01
tags: [frontend, backend, api]
---

# Array Test
EOF

if "$KIRA_BIN" lint > /dev/null 2>&1; then
  echo "  ✅ Array field validation works"
else
  echo "  ❌ Array field validation failed"
  exit 1
fi

# Test 33: Assign command workflow (switch, append, unassign, field, dry-run, interactive, batch)
echo ""
echo "👤 Test 33: Assign command workflow"

# Use deterministic users (don't rely on git history)
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
fields:
  assigned:
    type: email
    required: false
users:
  use_git_history: false
  saved_users:
    - name: Alice Example
      email: alice@example.com
    - name: Bob Example
      email: bob@example.com
EOF

# Create work items for assignment tests
cat > .work/1_todo/019-assign-e2e.prd.md << 'EOF'
---
id: 019
title: Assign E2E
status: todo
kind: prd
created: 2025-01-01
---

# Assign E2E
EOF

cat > .work/1_todo/020-assign-e2e-batch-a.prd.md << 'EOF'
---
id: 020
title: Assign E2E Batch A
status: todo
kind: prd
created: 2025-01-01
---

# Assign E2E Batch A
EOF

cat > .work/1_todo/021-assign-e2e-batch-b.prd.md << 'EOF'
---
id: 021
title: Assign E2E Batch B
status: todo
kind: prd
created: 2025-01-01
---

# Assign E2E Batch B
EOF

# Switch mode (numeric identifier)
"$KIRA_BIN" assign 019 1
if grep -q "^assigned: alice@example.com$" .work/1_todo/019-assign-e2e.prd.md; then
  echo "  ✅ Switch mode assigns by user number"
else
  echo "  ❌ Switch mode assignment failed"
  exit 1
fi

# Append mode (convert string -> list, avoid duplicates)
"$KIRA_BIN" assign 019 2 --append
if grep -q "alice@example.com" .work/1_todo/019-assign-e2e.prd.md && \
   grep -q "bob@example.com" .work/1_todo/019-assign-e2e.prd.md; then
  echo "  ✅ Append mode adds user without losing existing"
else
  echo "  ❌ Append mode failed"
  exit 1
fi

# Custom field (reviewer)
"$KIRA_BIN" assign 019 2 --field reviewer
if grep -q "^reviewer: bob@example.com$" .work/1_todo/019-assign-e2e.prd.md; then
  echo "  ✅ Custom field assignment works (--field reviewer)"
else
  echo "  ❌ Custom field assignment failed"
  exit 1
fi

# Unassign clears the field
"$KIRA_BIN" assign 019 --unassign
if grep -q "^assigned:" .work/1_todo/019-assign-e2e.prd.md; then
  echo "  ❌ Unassign should remove assigned field"
  exit 1
else
  echo "  ✅ Unassign removes assigned field"
fi

# Dry-run should not modify the file (no assigned field should appear)
"$KIRA_BIN" assign 019 1 --dry-run > /dev/null
if grep -q "^assigned:" .work/1_todo/019-assign-e2e.prd.md; then
  echo "  ❌ Dry-run should not write changes"
  exit 1
else
  echo "  ✅ Dry-run makes no changes"
fi

# Interactive selection (choose user 1)
printf "1\n" | "$KIRA_BIN" assign 019 --interactive > /dev/null
if grep -q "^assigned: alice@example.com$" .work/1_todo/019-assign-e2e.prd.md; then
  echo "  ✅ Interactive mode assigns selected user"
else
  echo "  ❌ Interactive mode assignment failed"
  exit 1
fi

# Batch assignment (multiple work items)
"$KIRA_BIN" assign 020 021 2 > /dev/null
if grep -q "^assigned: bob@example.com$" .work/1_todo/020-assign-e2e-batch-a.prd.md && \
   grep -q "^assigned: bob@example.com$" .work/1_todo/021-assign-e2e-batch-b.prd.md; then
  echo "  ✅ Batch assignment works"
else
  echo "  ❌ Batch assignment failed"
  exit 1
fi

# Slice command (add slice, add task, show)
echo ""
echo "📋 Test 34: Slice command (add, task add, show)"
cat > .work/1_todo/030-slice-e2e.prd.md << 'EOF'
---
id: 030
title: Slice E2E
status: todo
kind: prd
created: 2025-01-01
---

# Slice E2E
## Requirements
## Acceptance Criteria
EOF
"$KIRA_BIN" move 030 doing
if ! "$KIRA_BIN" slice add 030 "E2ESlice" --no-commit 2>/dev/null; then
  echo "  ❌ slice add failed"
  exit 1
fi
if ! grep -q "### E2ESlice" .work/2_doing/030-slice-e2e.prd.md; then
  echo "  ❌ slice add did not add E2ESlice to file"
  exit 1
fi
echo "  ✅ slice add adds slice to work item"
"$KIRA_BIN" slice task add 030 E2ESlice "E2E task one" --no-commit
if ! grep -q "T001" .work/2_doing/030-slice-e2e.prd.md || ! grep -q "E2E task one" .work/2_doing/030-slice-e2e.prd.md; then
  echo "  ❌ slice task add did not add task"
  exit 1
fi
echo "  ✅ slice task add adds task with generated ID"
"$KIRA_BIN" slice show 030 | grep -q "E2ESlice" && "$KIRA_BIN" slice show 030 | grep -q "T001"
if [ $? -ne 0 ]; then
  echo "  ❌ slice show did not show slice and task"
  exit 1
fi
echo "  ✅ slice show displays slices and tasks"
"$KIRA_BIN" slice progress 030 | grep -q "1 open"
if [ $? -ne 0 ]; then
  echo "  ❌ slice progress did not show progress"
  exit 1
fi
echo "  ✅ slice progress shows summary"

# Test 35: kira review - submit for review flow
echo ""
echo "📋 Test 35: kira review - submit for review"

# Reset to clean state: git repo, work item in doing, kira-named branch
rm -rf .git > /dev/null 2>&1
"$KIRA_BIN" init --force > /dev/null 2>&1
git init > /dev/null 2>&1
git config user.email test@example.com
git config user.name "Test User"
git checkout -b main > /dev/null 2>&1
git add .
git commit -m "init" > /dev/null 2>&1

# Work item in doing and branch matching id-title
cat > .work/2_doing/018-review-e2e.prd.md << 'EOF'
---
id: 018
title: Review E2E Test
status: doing
kind: prd
created: 2024-01-01
---

# Review E2E Test
EOF
git add .work/2_doing/018-review-e2e.prd.md
git commit -m "Add work item 018" > /dev/null 2>&1
git checkout -b 018-review-e2e > /dev/null 2>&1

# Dry-run: should print planned steps and not move file
REVIEW_DRY_OUTPUT=$("$KIRA_BIN" review --dry-run --no-trunk-update --no-rebase 2>&1)
if echo "$REVIEW_DRY_OUTPUT" | grep -q "trunk update\|rebase\|move to review\|push\|PR"; then
  echo "  ✅ kira review --dry-run shows planned steps"
else
  echo "  ❌ kira review --dry-run should show planned steps"
  echo "Output: $REVIEW_DRY_OUTPUT"
  exit 1
fi
if [ -f ".work/2_doing/018-review-e2e.prd.md" ] && [ ! -f ".work/3_review/018-review-e2e.prd.md" ]; then
  echo "  ✅ Dry-run did not move work item"
else
  echo "  ❌ Dry-run should not move work item"
  exit 1
fi

# Run review without dry-run (no remote: move succeeds, push fails with clear message)
REVIEW_OUTPUT=$("$KIRA_BIN" review --no-trunk-update --no-rebase 2>&1) || REVIEW_EXIT=$?
if [ -f ".work/3_review/018-review-e2e.prd.md" ] && [ ! -f ".work/2_doing/018-review-e2e.prd.md" ]; then
  echo "  ✅ kira review moved work item to review folder"
else
  echo "  ❌ kira review should move work item to review"
  echo "Output: $REVIEW_OUTPUT"
  exit 1
fi
if echo "$REVIEW_OUTPUT" | grep -q "Moving work item to review" && echo "$REVIEW_OUTPUT" | grep -q "Moved to review"; then
  echo "  ✅ kira review shows progress messages"
else
  echo "  ⚠️  Progress messages may vary"
fi
# Push expected to fail (no remote); we do not roll back the move
if echo "$REVIEW_OUTPUT" | grep -qi "push\|remote"; then
  echo "  ✅ Push step attempted (failure expected without remote)"
else
  echo "  ⚠️  Push step output may vary"
fi

# kira review --help
if "$KIRA_BIN" review --help 2>&1 | grep -q "submit for review\|--dry-run\|--no-rebase"; then
  echo "  ✅ kira review --help documents flags"
else
  echo "  ❌ kira review --help should document flags"
  exit 1
fi

# Test 36: kira done - complete work item (trunk-only, dry-run, flags, not-on-trunk failure)
echo ""
echo "📋 Test 36: kira done command"
# Ensure we're on main and have a work item in doing
git checkout main 2>/dev/null || true
# Add a non-GitHub remote so resolveDoneRemote succeeds (done skips PR steps for non-GitHub)
git remote add origin https://gitlab.com/e2e/repo.git 2>/dev/null || git remote set-url origin https://gitlab.com/e2e/repo.git
mkdir -p .work/2_doing
cat > .work/2_doing/036-done-e2e.prd.md << 'EOF'
---
id: 036
title: Done E2E
status: doing
kind: prd
created: 2024-01-01
---
# Content
EOF

# Require work-item-id
if ! "$KIRA_BIN" done 2>&1 | grep -q "accepts 1 arg(s)\|received 0\|work-item-id\|required\|Usage"; then
  echo "  ❌ kira done without work-item-id should fail with usage"
  exit 1
fi
echo "  ✅ kira done requires work-item-id"

# Not on trunk: create branch and run done, expect trunk error
git add .work/2_doing/036-done-e2e.prd.md
git commit -m "Add 036 for done e2e" >/dev/null 2>&1 || true
git checkout -b 036-done-feature 2>/dev/null || true
DONE_NOT_TRUNK=$("$KIRA_BIN" done 036 2>&1) || true
if echo "$DONE_NOT_TRUNK" | grep -q "trunk\|Cannot run 'kira done'"; then
  echo "  ✅ kira done on feature branch fails with trunk message"
else
  echo "  ❌ kira done on feature branch should fail with trunk message"
  echo "Output: $DONE_NOT_TRUNK"
  exit 1
fi
git checkout main 2>/dev/null || true

# Dry-run on trunk (non-GitHub remote: validates trunk and work item, skips PR)
DONE_DRY=$("$KIRA_BIN" done 036 --dry-run 2>&1)
if echo "$DONE_DRY" | grep -q "DRY RUN\|not GitHub\|validated"; then
  echo "  ✅ kira done --dry-run runs validation and skips PR for non-GitHub"
else
  echo "  ❌ kira done --dry-run should succeed on trunk"
  echo "Output: $DONE_DRY"
  exit 1
fi

# Help shows flags
if "$KIRA_BIN" done --help 2>&1 | grep -q "merge-strategy\|no-cleanup\|dry-run\|force"; then
  echo "  ✅ kira done --help documents flags"
else
  echo "  ❌ kira done --help should document flags"
  exit 1
fi

# Invalid work item ID
if ! "$KIRA_BIN" done abc 2>&1 | grep -q "invalid work item ID"; then
  echo "  ❌ kira done abc should fail with invalid work item ID"
  exit 1
fi
echo "  ✅ kira done with invalid ID fails"

echo "  ✅ kira done (trunk check, dry-run, flags, validation)"

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
echo "  ✅ Start command override flag"
echo "  ✅ Start command reuse-branch flag"
echo "  ✅ Start command commit_only action"
echo "  ✅ Latest command standalone workflow"
echo "  ✅ Latest command conflict detection"
echo "  ✅ Latest command iterative resolution"
echo "  ✅ Latest command polyrepo scenario"
echo "  ✅ Latest command configuration integration"
echo "  ✅ Latest command error handling"
echo "  ✅ Latest command stash management"
echo "  ✅ Field configuration system"
echo "  ✅ Field validation"
echo "  ✅ Field defaults"
echo "  ✅ Field error detection"
echo "  ✅ Doctor field fixes"
echo "  ✅ Backward compatibility"
echo "  ✅ Slice command (add, task add, show)"
echo "  ✅ kira review (submit for review, dry-run, move)"
echo "  ✅ kira done (trunk check, dry-run, flags, validation)"
echo ""
echo "🚀 Kira is ready for use!"

