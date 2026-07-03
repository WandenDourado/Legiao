# Metadata Format

Use JSON for generated sprite sheet metadata.

```json
{
  "image": "wizard_walk.png",
  "frame_width": 128,
  "frame_height": 192,
  "origin": "foot-center",
  "directions": ["S", "N", "E", "W", "SE", "SW", "NE", "NW"],
  "animations": {
    "walk": {
      "frames_per_direction": 6,
      "frame_time_seconds": 0.12,
      "rows": {
        "S": 0,
        "N": 1,
        "E": 2,
        "W": 3,
        "SE": 4,
        "SW": 5,
        "NE": 6,
        "NW": 7
      }
    }
  }
}
```

For the current Legiao player code, a compatibility export may use:

```json
{
  "frame_width": 165,
  "frame_height": 246,
  "directions": ["N", "S", "W", "SW", "NW"],
  "mirrored": {
    "E": "W",
    "SE": "SW",
    "NE": "NW"
  }
}
```
