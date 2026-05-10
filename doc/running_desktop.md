# Running Legiao on Desktop (Windows)

## Prerequisites
- **Go 1.22+** installed and in your `PATH`.
- **Raylib** development files (the Go bindings pull the pre‑compiled DLLs for Windows, but you need the `raylib.dll` in the working directory, which is already part of the repo).
- **Git** (to clone the repository).
- **Network** – ensure your firewall allows inbound TCP connections on the port you plan to use (default **9000**).

## Steps
1. **Clone the repository**
   ```bash
   git clone https://github.com/WandenDourado/Legiao.git
   cd Legiao
   ```
2. **Download dependencies**
   ```bash
   go mod tidy
   ```
3. **Build the desktop binary**
   ```bash
   go build -o legiao-desktop.exe ./cmd/desktop
   ```
   The output executable will be placed in the project root.
4. **Run the game**
   ```bash
   ./legiao-desktop.exe
   ```
   You should see a blue ball that can be moved with the joystick or arrow keys.

## Multiplayer (Wi‑Fi) on the Same Machine
To test networking locally:
1. **Start a host** – In the menu click **Host Game (Wi‑Fi)**. The server starts on port **9000** and registers itself as a player.
2. **Join as a client** – Open a second terminal and run the same executable. Click **Join Game (Wi‑Fi)**. The client connects to `127.0.0.1:9000`, sends its ID and color.
3. The host receives the join, registers the client, and broadcasts the state (including itself) to all peers.
4. **Result:** You will see all players (host + clients) with distinct colors. The host is now visible to all clients.
5. **Move players:** Each player can move independently and all peers will see the movement in real-time.

## Tips & Troubleshooting
- If the client cannot connect, verify that port **9000** is not blocked by Windows Defender or any other firewall.
- For remote testing on another device, replace `127.0.0.1` with the host’s LAN IP address (e.g., `192.168.1.42`).
- Logs are printed to the console; look for any `error` messages printed by the network layer.
