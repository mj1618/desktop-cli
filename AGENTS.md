# Agent Guide

Always keep README.md updated for users to understand the repo including how to install it.

Always keep SKILL.md updated with a minimal set of instructions for agents to make use of the installed tool.

## Saving Tips for Next Time

As you work through tasks, you'll discover useful patterns, workarounds, and tricks for interacting with specific applications. **Save these to `skills/{applicationName}.md`** (e.g. `skills/chrome.md`, `skills/safari.md`, `skills/notes.md`, `skills/finder.md`, `skills/system-settings.md`).

Before starting a task, check if a `skills/{appName}.md` file already exists and read it — it may save you time.

After completing (or attempting) a task, append any new tips you learned. Examples of things worth saving:
- The correct role/title to target for hard-to-find elements (e.g. "the address bar in Chrome is `AXTextField` titled `Address and search bar`")
- Non-obvious steps required to achieve something (e.g. "in Notes, you must click the note body before typing — clicking the title area first doesn't work")
- Accessibility tree quirks for specific apps (e.g. "Google Sheets cells aren't in the accessibility tree — use screenshot + coordinate clicking instead")
- Timing or wait requirements (e.g. "after opening System Settings, wait for the window before reading elements")
- Keyboard shortcuts that are faster than clicking through menus
- Element matching patterns that work reliably vs. ones that are brittle

Format each file as a simple list of tips:

```markdown
# {Application Name} Tips

- Tip 1
- Tip 2
```

Append new tips without removing existing ones. Keep tips concise and actionable.
