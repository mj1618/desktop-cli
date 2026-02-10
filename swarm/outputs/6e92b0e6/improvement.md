# Add --scope-id flag to read command

## Problem
When exploring complex UIs with many repeated elements (e.g., search results, tables, lists), the `read` command reads the entire UI tree, making it hard to isolate elements within a specific container.

For example, when reading Google Maps search results for "coffee shops near Sydney Opera House", I needed to find the first coffee shop name. The flat output contained 40+ text elements with "coffee" appearing multiple times, making it hard to identify which one was the first result name without carefully parsing the element hierarchy.

The `click` and `type` commands support `--scope-id INT` to limit text searches to descendants of a specific element ID, but `read` doesn't have this flag.

Command attempted:
```
desktop-cli read --app "Google Chrome" --scope-id 156 --depth 3 --format agent
```

Error:
```
Error: unknown flag: --scope-id
Usage:
  desktop-cli read [flags]
```

## Proposed Fix
Add `--scope-id INT` flag to the `read` command to limit element tree traversal to descendants of a specific element. This would work like:

```bash
desktop-cli read --app "Google Chrome" --scope-id 156 --depth 3
```

Output would only include elements that are descendants of element 156, making it much easier to inspect specific containers without reading the entire UI.

## Reproduction
1. Open Google Chrome and navigate to google.com/maps
2. Search for "coffee shops near Sydney Opera House"
3. Use `read --roles "lnk" --flat` to find a result container (e.g., id=160)
4. Try to read just that container:
   ```
   desktop-cli read --app "Google Chrome" --scope-id 160 --depth 3 --format agent
   ```
5. Observe the "unknown flag" error
