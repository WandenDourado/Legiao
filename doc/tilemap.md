# Tilemap

Use this doc for Tiled map files, tilesets, map loading, render layer order, and collision objects.

## Runtime Map

Runtime loads the map from:

```text
assets/maps/world_01.json
```

Maps and tilesets needed at runtime must live under `assets/` so Android can bundle them.

Current expected layout:

```text
assets/
  maps/
    world_01.json
    world_01.tmj
  tilesets/
    title_set.tsx
    title_set.png
```

## Loader Files

| File | Responsibility |
|---|---|
| `internal/tilemap/tilemap.go` | Tiled JSON/TMJ structs, map loading, tileset resolution. |
| `internal/tilemap/tsx.go` | External TSX parsing. |
| `internal/tilemap/renderer.go` | Tileset texture loading and layer rendering. |
| `internal/tilemap/collision.go` | Tiled object collision rectangles. |
| `internal/tilemap/filereader_desktop.go` | Desktop file reads. |
| `internal/tilemap/filereader_android.go` | Android APK asset reads via raylib. |

## Android Asset Rule

Do not use `os.ReadFile` directly for runtime map/tileset data. Use the tilemap `readFile()` abstraction so Android can read from APK assets.

Texture paths still go through `assets.Path()` before raylib loading.

## Tileset References

External tilesets should be referenced relative to the map file inside `assets/`, for example:

```json
{ "source": "../tilesets/title_set.tsx" }
```

TSX image paths are resolved relative to the TSX file directory.

## Rendering Order

`DrawFrame` uses `MapRenderer.DrawWithCamera` so map layers and entities share one camera block.

Current layer split:

- Draw lower map layers up to `objetcs`.
- Draw world entities.
- Draw upper layers from `objetcs`.

If map layer names change in Tiled, update the `DrawWithCamera` layer arguments in `internal/game/renderer.go`.

## Collision

Collision rectangles come from Tiled object layers through `tilemap.GetCollisionRects()`.

Player collision resolution is handled in `internal/game/collision.go`.

## Map Changes Checklist

1. Put runtime files under `assets/maps` and `assets/tilesets`.
2. Keep TSX/image references relative and Android-safe.
3. Verify map dimensions still produce valid `world.Bounds`.
4. Verify player spawn is inside bounds.
5. Verify layer names used by `DrawWithCamera`.
6. Run `go test ./...`.
