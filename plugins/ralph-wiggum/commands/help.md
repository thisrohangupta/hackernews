---
description: "Help and documentation for the Ralph Wiggum plugin"
---

# Ralph Wiggum Plugin Help

## Overview

The Ralph Wiggum technique is an iterative development methodology using continuous AI loops. The same prompt is fed to Claude repeatedly. The "self-referential" aspect comes from Claude seeing its own previous work in the files and git history, not from feeding output back as input.

## How It Works

Each iteration follows this cycle:
1. Claude receives the identical prompt
2. Modifies files to advance the task
3. Attempts to exit
4. A stop hook re-feeds the same prompt
5. Claude observes its previous changes
6. Continues refining until completion

The methodology is characterized as "deterministically bad in an undeterministic world" — predictable failures enable systematic improvement through prompt refinement.

## Main Commands

**`/ralph-loop <PROMPT> [OPTIONS]`** - Initiates a loop in your current session with options:
- `--max-iterations <n>` - Set maximum number of iterations (0 = unlimited)
- `--completion-promise <text>` - Text that signals task completion

**`/cancel-ralph`** - Terminates an active loop by removing the state file

## Completion Signals

Claude indicates completion using: `<promise>TASK COMPLETE</promise>`

The stop hook monitors for this tag. Without it or a max iteration limit, Ralph continues indefinitely.

## Ideal Use Cases

Best suited for:
- Well-defined tasks with clear success criteria
- Situations requiring iterative refinement
- Test-driven development workflows
- Tasks with automatic verification (tests, linters)

Less appropriate for:
- Subjective decisions
- Production debugging
- Tasks without clear completion criteria
