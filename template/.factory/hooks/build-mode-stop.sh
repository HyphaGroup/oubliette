#!/bin/bash
# Build Mode Stop Hook - prevents premature exit, handles verification, and advances through changes
# This hook is called by Factory Droid when an agent attempts to stop

set -euo pipefail

# Read hook input from stdin (includes transcript_path for output analysis)
HOOK_INPUT=$(cat)

# Check if we're in build mode (look for state file)
BUILD_STATE_FILE="$FACTORY_PROJECT_DIR/.factory/build-mode.json"
if [[ ! -f "$BUILD_STATE_FILE" ]]; then
  exit 0  # Not in build mode, allow exit
fi

# Get state from file
CHANGE_ID=$(jq -r '.change_id' "$BUILD_STATE_FILE")
BUILD_ALL=$(jq -r '.build_all // false' "$BUILD_STATE_FILE")
PHASE=$(jq -r '.phase // "build"' "$BUILD_STATE_FILE")
MAX_ITERATIONS=$(jq -r '.max_iterations // 100' "$BUILD_STATE_FILE")
ITERATION=$(jq -r '.iteration // 0' "$BUILD_STATE_FILE")
STARTED_AT=$(jq -r '.started_at // ""' "$BUILD_STATE_FILE")
TASKS_MD="$FACTORY_PROJECT_DIR/openspec/changes/$CHANGE_ID/tasks.md"

# Check iteration limit
if [[ $ITERATION -ge $MAX_ITERATIONS ]]; then
  echo "Build mode: max iterations ($MAX_ITERATIONS) reached" >&2
  rm "$BUILD_STATE_FILE"
  exit 0  # Allow exit
fi

# Safety check: if stop_hook_active is true, we're already in a re-prompt loop
STOP_HOOK_ACTIVE=$(echo "$HOOK_INPUT" | jq -r '.stop_hook_active // false')
if [[ "$STOP_HOOK_ACTIVE" == "true" && $ITERATION -gt 50 ]]; then
  echo "Build mode: stop_hook_active detected after 50+ iterations, allowing exit" >&2
  rm "$BUILD_STATE_FILE"
  exit 0
fi

# Helper: update state file
update_state() {
  local updates="$1"
  jq "$updates" "$BUILD_STATE_FILE" > "${BUILD_STATE_FILE}.tmp"
  mv "${BUILD_STATE_FILE}.tmp" "$BUILD_STATE_FILE"
}

# Helper: get last assistant output from transcript
get_last_output() {
  local transcript_path
  transcript_path=$(echo "$HOOK_INPUT" | jq -r '.transcript_path // empty')
  if [[ -n "$transcript_path" && -f "$transcript_path" ]]; then
    grep '"role":"assistant"' "$transcript_path" | tail -1 | jq -r '
      .message.content | map(select(.type == "text")) | map(.text) | join("\n")
    ' 2>/dev/null || echo ""
  fi
}

# Helper: check if tasks.md is stale (not modified since build started)
tasks_stale() {
  if [[ -z "$STARTED_AT" || ! -f "$TASKS_MD" ]]; then
    return 1  # Can't determine, assume not stale
  fi
  
  # Parse ISO8601 timestamp to epoch (macOS compatible)
  local started_epoch tasks_mtime
  started_epoch=$(date -j -f "%Y-%m-%dT%H:%M:%S" "${STARTED_AT%.*}" +%s 2>/dev/null || echo 0)
  tasks_mtime=$(stat -f %m "$TASKS_MD" 2>/dev/null || echo 0)
  
  [[ $tasks_mtime -lt $started_epoch ]]
}

# Helper: get next incomplete change
get_next_change() {
  # Source nvm for openspec command
  source ~/.nvm/nvm.sh 2>/dev/null || true
  
  openspec list --json 2>/dev/null | jq -r '
    .changes | map(select(.completedTasks < .totalTasks)) | .[0].name // empty
  '
}

# Helper: advance to next change or exit
advance_or_exit() {
  if [[ "$BUILD_ALL" == "true" ]]; then
    local next_change
    next_change=$(get_next_change)
    
    if [[ -n "$next_change" && "$next_change" != "$CHANGE_ID" ]]; then
      update_state ".change_id = \"$next_change\" | .phase = \"build\" | .iteration = 0"
      jq -n --arg change "$next_change" '{
        "decision": "block",
        "reason": "/openspec-apply \($change)"
      }'
      exit 0
    fi
  fi
  
  # No more changes - allow exit
  rm "$BUILD_STATE_FILE"
  exit 0
}

# Source nvm for openspec commands
source ~/.nvm/nvm.sh 2>/dev/null || true

# PHASE: BUILD - check task completion
if [[ "$PHASE" == "build" ]]; then
  # Get task state from openspec
  TASK_OUTPUT=$(openspec instructions apply --change "$CHANGE_ID" --json 2>/dev/null || echo '{"state":"unknown"}')
  TASK_STATE=$(echo "$TASK_OUTPUT" | jq -r '.state // "unknown"')
  
  if [[ "$TASK_STATE" == "all_done" ]]; then
    # Check if tasks.md is stale (agent may have lied about completion)
    if tasks_stale; then
      update_state ".iteration = $((ITERATION + 1))"
      jq -n '{
        "decision": "block",
        "reason": "Agent is idle but tasks.md has not been updated. Please update tasks.md to mark completed tasks with [x] before continuing."
      }'
      exit 0
    fi
    
    # Tasks complete - transition to verify phase
    update_state '.phase = "verify" | .iteration = 0'
    
    VERIFY_PROMPT="All tasks for change \"$CHANGE_ID\" are complete. Before archiving:

1. Run the project's build/compile step (check Makefile, build.sh, package.json, Cargo.toml, go.mod, etc.)
2. Run the project's test suite
3. Run any linters/typecheckers configured for the project (check AGENTS.md for guidance)
4. Fix any errors you find

When all builds pass and tests are green, output exactly: VERIFIED"

    jq -n --arg prompt "$VERIFY_PROMPT" '{
      "decision": "block",
      "reason": $prompt
    }'
    exit 0
  fi
  
  # Tasks remaining - re-prompt to continue
  COMPLETE=$(echo "$TASK_OUTPUT" | jq -r '.progress.complete // 0')
  TOTAL=$(echo "$TASK_OUTPUT" | jq -r '.progress.total // 0')
  PROGRESS="$COMPLETE/$TOTAL"
  
  update_state ".iteration = $((ITERATION + 1))"
  
  jq -n --arg progress "$PROGRESS" --arg change "$CHANGE_ID" '{
    "decision": "block",
    "reason": "Tasks remaining: \($progress). Continue working on change: \($change). Check tasks.md for incomplete items marked with [ ] and keep implementing."
  }'
  exit 0
fi

# PHASE: VERIFY - check for VERIFIED marker
if [[ "$PHASE" == "verify" ]]; then
  LAST_OUTPUT=$(get_last_output)
  
  if echo "$LAST_OUTPUT" | grep -q "VERIFIED"; then
    # Verification passed - archive and advance
    openspec archive "$CHANGE_ID" 2>/dev/null || true
    git add -A 2>/dev/null || true
    git commit -m "feat($CHANGE_ID): implementation complete" 2>/dev/null || true
    
    advance_or_exit
  fi
  
  # Not verified yet - re-prompt
  update_state ".iteration = $((ITERATION + 1))"
  jq -n --arg change "$CHANGE_ID" '{
    "decision": "block",
    "reason": "Verification not complete for change: \($change). Please ensure build passes, tests are green, and output VERIFIED when done."
  }'
  exit 0
fi

# Unknown phase - allow exit
exit 0
