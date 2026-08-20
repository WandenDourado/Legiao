# Documentation Index

Read only the document that matches the task. `AGENTS.md` is the required agent entrypoint and contains the routing table.

## Current Source Of Truth

| Area | Doc |
|---|---|
| Repository/package map | `project_structure.md` |
| Coding rules and ownership | `coding_patterns.md` |
| Android build and APK assets | `android.md` |
| Input controls, sprint, aim | `controls.md` |
| Cadencia de ataque, cooldown, morte, game over | `combat_rules.md` |
| Multiplayer, discovery, sync | `network.md` |
| Custo de render/VRAM/banda e plano de otimizacao | `performance.md` |
| Diálogo narrativo e pausa de cena | `dialogue.md` |
| Camera and world bounds | `camera.md` |
| Tilemap loading/rendering/collision | `tilemap.md` |
| Village tileset module contracts | `tileset_spec.md` |
| Visual style for new assets (palette, luz, escala) | `art_style.md` |
| Desktop build/run | `running_desktop.md` |
| Game scope | `overview.md` |

## History

`changelog.md` is append-only history. Do not read it by default; use it only to investigate regressions or add a new entry.

## Removed Docs

The old fix journals `android_support.md`, `mobile_input.md`, `networking_fix.md`, and `udp_discovery.md` were merged into the source-of-truth docs above. Future fixes should update the area doc and append one concise changelog entry instead of creating a new long fix document.
