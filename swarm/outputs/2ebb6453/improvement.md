# Ambiguous match error should show more element context

## Problem

When multiple elements match a text query, the error message only shows element ID and role, making it impossible to distinguish between them without additional reads:

```bash
$ desktop-cli click --text "New Note" --app "Notes"
Error: multiple elements match text "New Note" — use --id, --exact, or --scope-id to narrow:
  id=50 role=txt
  id=63 role=txt
```

This requires an additional `read` command to inspect elements 50 and 63 to understand which one to click, adding unnecessary round-trips.

## Proposed Fix

The ambiguous match error should include enough context to distinguish elements at a glance:

```bash
Error: multiple elements match text "New Note" — use --id, --exact, or --scope-id to narrow:
  id=50 role=txt bounds=[1035,452,65,18] path="window > group > scroll > list > row > cell > cell > txt"
  id=63 role=txt bounds=[1035,568,65,18] path="window > group > scroll > list > row > cell > cell > txt"
```

Or in a more compact format:

```bash
Error: multiple elements match text "New Note" — use --id, --exact, or --scope-id to narrow:
  id=50 txt (1035,452,65,18) "window > ... > row > cell > txt"
  id=63 txt (1035,568,65,18) "window > ... > row > cell > txt"
```

At minimum, include bounds/coordinates so agents can see spatial relationships. Ideally also include abbreviated path breadcrumbs to show context.

## Reproduction

1. Open Notes app with multiple notes titled "New Note"
2. Run: `desktop-cli click --text "New Note" --app "Notes"`
3. Observe error shows only id and role, no distinguishing information
