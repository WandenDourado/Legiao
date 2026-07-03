# Controls

Use this doc for keyboard, mouse, touch joystick, sprint, and aim behavior.

## Desktop

| Action | Input |
|---|---|
| Move | WASD or arrow keys |
| Sprint | Left Shift |
| Aim | Mouse position in world space |
| Fire | Left mouse button |

Desktop mouse position is screen-space and must be converted before attack targeting:

```go
screenPos := rl.GetMousePosition()
worldPos := rl.GetScreenToWorld2D(screenPos, cam.Camera)
```

## Android

Android controls are screen-relative and computed each frame in `game.ProcessInput`.

| Control | Position |
|---|---|
| Movement joystick | center at `screenWidth * 0.15`, `screenHeight * 0.80` |
| Aim/action joystick | center at `screenWidth * 0.85`, `screenHeight * 0.80` |
| Base radius | `screenHeight * 0.08` |

`internal/input/touch.go` tracks stable touch IDs so movement and attack touches do not steal each other.

## Sprint

| Platform | Trigger |
|---|---|
| Desktop | Left Shift |
| Android | Movement joystick magnitude greater than `entity.SprintThreshold` (`0.70`) |

Sprint currently changes animation playback speed, not the movement speed constant.

## Aim And Fire

`input.AimJoystick` is a dual-function action control:

- Quick tap: fires in the default/current direction.
- Press and drag: updates `AimDir`.
- Release: fires in the aimed direction.

The aim direction is converted to a world-space target relative to the player:

```go
targetX := p.Position.X + aimDir.X*100
targetY := p.Position.Y + aimDir.Y*100
```

Both desktop and Android send attacks through the same network path: clients send `MsgAttack`; the host creates projectiles.
