# Running On Desktop

## Build

From the repo root:

```powershell
go build -o legiao-desktop.exe ./cmd/desktop
```

## Run

```powershell
.\legiao-desktop.exe
```

The desktop entrypoint calls `game.Run(game.DefaultConfig())`.

## Local Multiplayer Test

1. Start one desktop instance and click `Host Game (Wi-Fi)`.
2. Start a second instance and click `Join Game (Wi-Fi)`.
3. Pick the discovered host or use `Manual IP` with `127.0.0.1`.

For another device on the LAN, use the host LAN IP if discovery fails. Ensure Windows Firewall allows inbound TCP port `9000`.

## Controls

See `controls.md`.
