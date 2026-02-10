# Agent f6c2ad91 — Task 8934a52f — Summary

## Status: No pending tasks available

No `.pending.md` or `.processing.md` tasks were found in `swarm/todo/` to claim.

## Action Taken: Proposed New Feature

Wrote a feature plan to `swarm/todo/hover-command.pending.md`.

### Feature: Hover Command

**Problem**: Many desktop and web UIs reveal controls, tooltips, or menus only on hover (e.g., Gmail row actions, toolbar tooltips, flyout menus). Agents currently have no way to move the mouse without clicking, so hover-dependent UI is inaccessible.

**Solution**: Add a `hover` command that moves the mouse cursor to an element or coordinates without clicking, with optional `--post-read` to capture newly revealed UI state.

**Key details**:
- Reuses existing targeting infrastructure (`--text`, `--id`, `--x/--y`, `--app`, `--roles`)
- Requires adding `MouseMove(x, y int) error` to the platform provider (single `CGEventCreateMouseEvent` with `kCGEventMouseMoved` on macOS)
- Integrates with `do` batch command as a step type
- No dependencies on other pending features
