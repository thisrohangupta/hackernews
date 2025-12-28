#!/bin/bash
# Ralph Wiggum Stop Hook
# Prevents session exit when a ralph-loop is active and feeds the prompt back to Claude

STATE_FILE=".claude/ralph-loop.local.md"

# Exit early if no active loop
if [[ ! -f "$STATE_FILE" ]]; then
  echo '{"decision": "allow"}'
  exit 0
fi

# Parse YAML frontmatter from state file
parse_frontmatter() {
  local key="$1"
  grep "^${key}:" "$STATE_FILE" | sed "s/^${key}: *//" | tr -d '"'
}

# Read state
ACTIVE=$(parse_frontmatter "active")
ITERATION=$(parse_frontmatter "iteration")
MAX_ITERATIONS=$(parse_frontmatter "max_iterations")
COMPLETION_PROMISE=$(parse_frontmatter "completion_promise")
ORIGINAL_PROMPT=$(parse_frontmatter "prompt")

# Validate state
if [[ "$ACTIVE" != "true" ]]; then
  echo '{"decision": "allow"}'
  exit 0
fi

# Validate numeric fields
if ! [[ "$ITERATION" =~ ^[0-9]+$ ]]; then
  echo "Warning: Invalid iteration count in state file" >&2
  ITERATION=1
fi

if ! [[ "$MAX_ITERATIONS" =~ ^[0-9]+$ ]]; then
  MAX_ITERATIONS=0
fi

# Check max iterations
if [[ "$MAX_ITERATIONS" -gt 0 && "$ITERATION" -ge "$MAX_ITERATIONS" ]]; then
  rm -f "$STATE_FILE"
  echo '{"decision": "allow"}'
  exit 0
fi

# Check for completion promise in transcript
if [[ -n "$COMPLETION_PROMISE" && -n "$CLAUDE_TRANSCRIPT" && -f "$CLAUDE_TRANSCRIPT" ]]; then
  # Get last assistant message from JSONL transcript
  LAST_MESSAGE=$(tail -20 "$CLAUDE_TRANSCRIPT" | grep '"role":"assistant"' | tail -1)

  # Check for promise tags
  if echo "$LAST_MESSAGE" | grep -q "<promise>.*${COMPLETION_PROMISE}.*</promise>"; then
    rm -f "$STATE_FILE"
    echo '{"decision": "allow"}'
    exit 0
  fi
fi

# Increment iteration
NEW_ITERATION=$((ITERATION + 1))
sed -i "s/^iteration: .*/iteration: $NEW_ITERATION/" "$STATE_FILE"

# Build system message
SYSTEM_MSG="[Ralph Loop - Iteration $NEW_ITERATION"
if [[ "$MAX_ITERATIONS" -gt 0 ]]; then
  SYSTEM_MSG+=" of $MAX_ITERATIONS"
fi
SYSTEM_MSG+="]"

if [[ -n "$COMPLETION_PROMISE" ]]; then
  SYSTEM_MSG+=" Complete the task. When done, output: <promise>$COMPLETION_PROMISE</promise>"
fi

# Block exit and feed prompt back
cat <<EOF
{
  "decision": "block",
  "reason": "$SYSTEM_MSG\n\n$ORIGINAL_PROMPT"
}
EOF
