# Google Sheets cells not exposed in accessibility tree

## Issue
Google Sheets cells and their contents are not exposed through the macOS accessibility API in a way that allows desktop-cli to read or interact with them. This prevents verification of cell values, formulas, and results.

## Details
- Successfully navigated to Google Drive and created a new blank spreadsheet
- Successfully typed 5 different formulas using `desktop-cli type`:
  1. `=SUM(1,2,3,4,5)` in cell A1
  2. `=AVERAGE(10,20,30,40,50)` in cell A2
  3. `=COUNT(A1:A2)` in cell A3
  4. `=IF(A1>10,"Greater","Less")` in cell A4
  5. `=CONCATENATE("Result: ",A2)` in cell A5
- After entering formulas, attempted to verify they were stored correctly
- Attempted multiple read commands to verify cell contents:
  - `read --depth 3 --roles "cell,input,txt"` - returned no elements
  - `read --depth 6 --roles "cell,input,txt"` - returned no elements
  - `read --depth 5 | grep` for formula keywords - no matches
  - `read --focused` - shows only a hidden input element at [78, 125, 0, 1] with value "|4+" (appears to be internal state, not cell content)

## Impact
- Cannot verify that formulas were successfully saved
- Cannot read cell values or formula results
- Cannot use desktop-cli to interact with Google Sheets in any meaningful way beyond typing
- The spreadsheet grid cells are not exposed as individual accessible elements
- There is only a single hidden/internal input element exposed

## Root Cause
Google Sheets likely uses a custom rendering engine (similar to Gmail's contenteditable divs) that doesn't properly expose individual cell values and formulas through the macOS accessibility API. The accessibility tree shows only the application shell, not the actual spreadsheet data.

## Workaround
Currently none. Screenshot with vision model fallback is mentioned in the docs but requires screen recording permission.

## Suggestion
This may be a limitation of Google Sheets' accessibility implementation rather than desktop-cli itself. However, it severely limits the tool's usefulness for interacting with web-based spreadsheet applications. Consider:
1. Documenting this as a known limitation for Google Sheets
2. Investigating if other spreadsheet apps (Excel Online, LibreOffice) have similar issues
3. Potentially implementing a web-based version that can directly inject into the DOM for better access
