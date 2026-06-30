# Changelog

All notable changes to pzmod are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project follows
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [3.0.0]

pzmod v3 is a ground-up rewrite. It replaces the old prompt-driven flow with a
full terminal app, adds a scriptable CLI over the same service layer, and introduces
byte-exact `servertest.ini` handling.

### Added

- **Terminal app:** a keyboard-driven Bubble Tea UI with a dashboard, installed-mods
  view, search, load-order editor, validation, backups, server info, and settings.
- **Workshop search & browse:** find mods by name and read their details in-app,
  without copying IDs from the browser.
- **Dependency auto-resolution:** required items (and their dependencies) are pulled
  in automatically; collections are expanded for their members.
- **Load-order management:** reorder mods by hand, or apply a framework-first
  suggestion that loads shared libraries ahead of the mods that need them.
- **Type-to-filter:** press `/` on any long list (installed mods, search results,
  load order) to narrow it live.
- **Dry-run validation:** flags missing dependencies, unknown mod IDs, delisted or
  banned items, duplicate IDs, ModID clashes, bad map order, and Build 42
  compatibility, with one-key fixes where possible. `pzmod validate` exits non-zero
  on errors for CI use.
- **Backups & rollback:** every save takes a timestamped, byte-exact snapshot first;
  restore any previous version in one step.
- **Multiple server profiles:** manage several servers, each with its own config
  path, build, and backups.
- **Build 41 and Build 42 awareness:** first-class support for both, including the
  Build 42 `\ModID` and `WorkshopID\ModID` reference formats (matched and validated
  by logical mod ID), plus per-profile compatibility hints.
- **Scriptable CLI:** `profile`, `validate`, `search`, `backup`, `mods`
  (`--resolve-deps`), `get`, `set`, `copy`, `api-key`, and `update`.

### Changed

- Configuration now lives under your OS config directory
  (`~/.config/pzmod`, or `%AppData%\pzmod` on Windows) instead of `~/.pzmod`.
- Inline comments in `servertest.ini` are now preserved in place, and blank-line
  structure and line endings are kept exactly; only values you change are rewritten.
- Scriptable commands accept `--file`, `--profile`, or fall back to the default
  profile (the previous `--file` behaviour still works).
- Mods and WorkshopItems are treated as independent lists (the load order and the
  download set), rather than assuming a strict 1:1 mapping.

### Migration from v2

- Your existing `~/.pzmod` Steam API key is migrated automatically on first run;
  the old file is left untouched.
- The `get`, `set`, `copy`, `api-key`, and `update` commands continue to work as
  before, so existing scripts keep running.

[3.0.0]: https://github.com/kldzj/pzmod/releases/tag/v3.0.0
