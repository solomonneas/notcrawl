# Changelog

## Unreleased

- Add crawlkit control metadata/status surfaces with `metadata --json`, `status --json`, and `doctor --json`.
- Report primary archive and desktop-cache SQLite inventories in status JSON for shared local control surfaces.
- Add `notcrawl tui`, a local terminal browser for archived pages and databases backed by `crawlkit/tui`.
- Render TUI rows with compact panes so page and database metadata stays in context/detail instead of crowding the row list.
- Resolve database parent names for the TUI parent pane so collection nesting is readable instead of raw IDs.
- Route the TUI through read-only SQLite access and cover the JSON fallback in tests.
