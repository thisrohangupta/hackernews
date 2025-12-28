# Ralph Wiggum Plugin

Implementation of the Ralph Wiggum technique - continuous self-referential AI loops for interactive iterative development.

## Overview

Ralph is essentially "a Bash loop" that repeatedly feeds prompts to Claude, enabling self-improvement through iteration. Named after the Simpsons character, it embodies persistent refinement despite obstacles.

The same prompt is fed to Claude repeatedly. The "self-referential" aspect comes from Claude seeing its own previous work in the files and git history, not from feeding output back as input.

## How It Works

1. Claude receives the identical prompt each iteration
2. Modifies files to advance the task
3. Attempts to exit the session
4. A stop hook intercepts and re-feeds the same prompt
5. Claude observes its previous changes in files
6. Continues refining until completion criteria are met

## Commands

### `/ralph-loop <PROMPT> [OPTIONS]`

Initiates a loop in your current session.

**Options:**
- `--max-iterations <n>` - Maximum number of iterations (0 = unlimited)
- `--completion-promise <text>` - Text that signals task completion

**Example:**
```
/ralph-loop "Build a REST API with full test coverage" --completion-promise "All tests pass" --max-iterations 20
```

### `/cancel-ralph`

Terminates an active loop by removing the state file.

## Completion Signals

Claude indicates completion using promise tags:
```
<promise>TASK COMPLETE</promise>
```

The stop hook monitors for this tag. Without it or a max iteration limit, Ralph continues indefinitely.

## Best Practices

Effective prompts require:
- **Clear completion criteria** with specific success conditions
- **Incremental phases** breaking large tasks into manageable steps
- **Self-correction mechanisms** like test-driven development workflows
- **Iteration limits** as safety mechanisms preventing infinite loops

## When to Use

Ralph excels with:
- Well-defined tasks with clear success metrics
- Automatic verification (tests, linters, type checkers)
- Iterative refinement tasks

Less suitable for:
- Tasks requiring subjective human judgment
- Unclear or evolving requirements
- Production debugging

## File Structure

```
ralph-wiggum/
├── .claude-plugin/
│   └── plugin.json        # Plugin metadata
├── commands/
│   ├── cancel-ralph.md    # Cancel loop command
│   ├── help.md            # Plugin help documentation
│   └── ralph-loop.md      # Start loop command
├── hooks/
│   ├── hooks.json         # Hook configuration
│   └── stop-hook.sh       # Stop hook implementation
├── scripts/
│   └── setup-ralph-loop.sh # Loop initialization script
└── README.md              # This file
```
