# Window titles always empty in `list` command

## Summary

The `desktop-cli list --windows` and `desktop-cli list --app "Google Chrome"` commands return empty strings for the `title` field of every window, even when the windows have titles that are accessible through the `read` command.

## Steps to reproduce

1. Open Google Chrome with Gmail loaded
2. Run: `desktop-cli list --app "Google Chrome"`
3. Observe `title: ""` in the output
4. Run: `desktop-cli read --app "Google Chrome" --depth 2 --flat`
5. Observe the window element (role: window) has the correct title, e.g. `t: "Inbox (23,264) - matthew.stephen.james@gmail.com - Gmail - Google Chrome"`

## Expected behavior

```yaml
- app: Google Chrome
  pid: 44037
  title: "Inbox (23,264) - matthew.stephen.james@gmail.com - Gmail - Google Chrome"
  id: 178312
  bounds: [77, 38, 1644, 1079]
  focused: true
```

## Actual behavior

```yaml
- app: Google Chrome
  pid: 44037
  title: ""
  id: 178312
  bounds: [77, 38, 1644, 1079]
  focused: true
```

## Impact

- Cannot identify windows by title without doing a full `read` for each one
- The `--window "title"` flag on other commands may not work if it relies on the same title-fetching logic
- Makes it harder for agents to quickly find the right window to interact with

## Notes

This affected all applications observed during testing, not just Chrome. Every window from Cursor, Claude, Code, Finder, iTerm2, Spotify, etc. all showed `title: ""`.
