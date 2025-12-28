---
description: "Start a Ralph Wiggum iterative loop"
allowed-tools: ["Bash"]
---

# Start Ralph Loop

```!
"${CLAUDE_PLUGIN_ROOT}/scripts/setup-ralph-loop.sh" $ARGUMENTS
```

Check the output above for the state file location.

If a completion promise was set, read it from `.claude/ralph-loop.local.md` and note:

**CRITICAL**: When your completion promise becomes TRUE, you MUST output it wrapped in `<promise></promise>` tags.

For example, if your completion promise is "All tests pass", when tests genuinely pass, output:
```
<promise>All tests pass</promise>
```

**IMPORTANT**: The promise statement MUST be completely and unequivocally TRUE when you output it. Never output a false promise to exit prematurely, even when facing apparent obstacles or lengthy iterations.

Now proceed with the task described in the prompt.
