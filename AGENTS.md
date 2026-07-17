# AGENTS.md - Legiao

Single entry point for AI agents. Read this file first. Do not read every doc by default; use the routing table and open only the docs that match the task.

## Minimal Workflow

1. Read this file.
2. Check `git status --short`.
3. Read only the area docs listed below for the task.
4. Keep changes scoped; do not rewrite unrelated files.
5. Update affected docs and append one line to `doc/changelog.md` when behavior, architecture, build, or workflow changes.

## Hard Rules

- Never add game logic directly to `internal/game/loop.go`; it is orchestration only.
- Never pass raw paths to `rl.Load*`; use `assets.Path()`.
- Keep each `.go` file to one responsibility. Split files before they grow beyond roughly 150 lines unless they are pure orchestration.
- Use build tags for platform behavior (`//go:build android` / `//go:build !android`), not runtime `if isAndroid` checks.
- Use `world.Bounds` or map-derived dimensions for world logic. Do not use screen constants for camera, projectile, spawn, or collision bounds.
- Do not run destructive cleanup, commit, push, publish, or upload artifacts unless explicitly requested.

## Docs Routing

| Task touches | Read |
|---|---|
| Documentation map or repo layout | `doc/README.md`, `doc/project_structure.md` |
| Code style, package ownership, file split decisions | `doc/coding_patterns.md` |
| Android build, APK/AAB, APK assets | `doc/android.md` |
| Input, joystick, aim, sprint | `doc/controls.md` |
| Multiplayer, host/client, discovery, combat/projectile sync | `doc/network.md` |
| Camera, map bounds, world-space rendering | `doc/camera.md` |
| Tiled maps, tilesets, collision layers | `doc/tilemap.md` |
| Desktop build/run | `doc/running_desktop.md` |
| Product scope or feature intent | `doc/overview.md` |

`doc/changelog.md` is history, not prerequisite reading. Use it only when investigating regressions or appending a new entry.

## Skills

### create-character-sprites

Skill file: `skills/create-character-sprites/SKILL.md`

Use for reference-driven character sprite work:
- Creating character bibles, model sheets, directional animation frames, sprite sheets, and metadata.
- Validating RPG/isometric/top-down character sprite frames before game integration.

Generate and validate one direction at a time. Do not use for Android artifact generation unless the task also asks to bundle or build Android outputs.

### install-character-sprites

Skill file: `skills/install-character-sprites/SKILL.md`

Use to add a validated output from `create-character-sprites` to Legiao as a selectable playable character. It registers the asset, preserves the shared animation contract, and verifies desktop compilation.

### legiao-android-build

Skill file: `skills/legiao-android-build/SKILL.md`

Use only for Android artifact work:
- Building debug APKs or release AABs.
- Preparing or verifying Android release artifacts.

Do not use for desktop-only work, docs-only work, gameplay changes, UI changes, networking changes, or code review without Android artifact generation.

Prerequisites: Go, Android SDK/NDK, and Gradle wrapper available. Release builds require signing environment variables; never print secret values.
