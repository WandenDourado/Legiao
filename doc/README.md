# Documentation Index

Read only the document that matches the task. `AGENTS.md` is the required agent entrypoint and contains the routing table.

## Current Source Of Truth

| Area | Doc |
|---|---|
| Repository/package map | `project_structure.md` |
| Coding rules and ownership | `coding_patterns.md` |
| Android build and APK assets | `android.md` |
| Input controls, sprint, aim | `controls.md` |
| Multiplayer, discovery, sync | `network.md` |
| Camera and world bounds | `camera.md` |
| Tilemap loading/rendering/collision | `tilemap.md` |
| Desktop build/run | `running_desktop.md` |
| Game scope | `overview.md` |

## History

`changelog.md` is append-only history. Do not read it by default; use it only to investigate regressions or add a new entry.

## Removed Docs

The old fix journals `android_support.md`, `mobile_input.md`, `networking_fix.md`, and `udp_discovery.md` were merged into the source-of-truth docs above. Future fixes should update the area doc and append one concise changelog entry instead of creating a new long fix document.
