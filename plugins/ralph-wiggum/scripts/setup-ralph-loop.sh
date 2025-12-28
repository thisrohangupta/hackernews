#!/bin/bash
# Ralph Loop Setup Script
# Initializes a self-referential development loop

set -e

STATE_FILE=".claude/ralph-loop.local.md"
MAX_ITERATIONS=0
COMPLETION_PROMISE=""
PROMPT=""

# Display help
show_help() {
  cat <<EOF
Ralph Loop - Iterative AI Development

Usage: /ralph-loop <PROMPT> [OPTIONS]

Options:
  --max-iterations <n>        Maximum iterations (0 = unlimited, default: 0)
  --completion-promise <text> Text that signals task completion
  -h, --help                  Show this help message

Examples:
  /ralph-loop "Build a todo API" --completion-promise "All tests pass" --max-iterations 20
  /ralph-loop "Fix all linting errors" --max-iterations 10

WARNING: Without --max-iterations or --completion-promise, Ralph runs infinitely!

Monitor progress: cat $STATE_FILE
Cancel loop: /cancel-ralph
EOF
}

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    -h|--help)
      show_help
      exit 0
      ;;
    --max-iterations)
      if [[ -z "$2" || ! "$2" =~ ^[0-9]+$ ]]; then
        echo "Error: --max-iterations requires a numeric value" >&2
        exit 1
      fi
      MAX_ITERATIONS="$2"
      shift 2
      ;;
    --completion-promise)
      if [[ -z "$2" ]]; then
        echo "Error: --completion-promise requires a value" >&2
        exit 1
      fi
      COMPLETION_PROMISE="$2"
      shift 2
      ;;
    *)
      # Collect remaining args as prompt
      if [[ -n "$PROMPT" ]]; then
        PROMPT="$PROMPT $1"
      else
        PROMPT="$1"
      fi
      shift
      ;;
  esac
done

# Validate prompt
if [[ -z "$PROMPT" ]]; then
  echo "Error: A prompt is required" >&2
  echo "Usage: /ralph-loop <PROMPT> [OPTIONS]" >&2
  echo "Run with --help for more information" >&2
  exit 1
fi

# Check for existing loop
if [[ -f "$STATE_FILE" ]]; then
  echo "Warning: An active Ralph loop already exists!" >&2
  echo "Use /cancel-ralph to stop it first, or remove $STATE_FILE manually" >&2
  exit 1
fi

# Ensure .claude directory exists
mkdir -p .claude

# Create state file
cat > "$STATE_FILE" <<EOF
---
active: true
iteration: 1
max_iterations: $MAX_ITERATIONS
completion_promise: "$COMPLETION_PROMISE"
prompt: "$PROMPT"
started_at: $(date -Iseconds)
---

# Ralph Loop State

This file tracks the state of an active Ralph loop.
Delete this file or run /cancel-ralph to stop the loop.
EOF

echo "Ralph loop initialized!"
echo "State file: $STATE_FILE"
echo "Prompt: $PROMPT"
if [[ "$MAX_ITERATIONS" -gt 0 ]]; then
  echo "Max iterations: $MAX_ITERATIONS"
else
  echo "Max iterations: unlimited"
fi
if [[ -n "$COMPLETION_PROMISE" ]]; then
  echo "Completion promise: $COMPLETION_PROMISE"
fi
echo ""
echo "The loop will continue until:"
if [[ "$MAX_ITERATIONS" -gt 0 ]]; then
  echo "  - Max iterations ($MAX_ITERATIONS) reached"
fi
if [[ -n "$COMPLETION_PROMISE" ]]; then
  echo "  - Completion promise output: <promise>$COMPLETION_PROMISE</promise>"
fi
echo "  - Loop manually cancelled via /cancel-ralph"
