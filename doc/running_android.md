# Running Legiao on Android

## Prerequisites

- **Go 1.22+** installed and in your `PATH`.
- **Android SDK** - Set `ANDROID_HOME` environment variable.
- **Android NDK r23** - Set `ANDROID_NDK_HOME` environment variable.
- **Gradle** - Available via `gradlew.bat` in the project.
- **Network permissions** - Already included in the manifest for multiplayer.

## Directory Structure

```
cmd/android/build/
├── main.go              # Android main entry point (uses game.Run())
├── main_android.go       # Sets up init() with rl.SetCallbackFunc(main)
├── android/
│   ├── AndroidManifest.xml   # App manifest (includes INTERNET permission)
│   ├── build.gradle        # Android build config
│   └── res/                  # Icons and splash screens
├── androidcompile.bat      # Compile Go → .so libraries
├── build.gradle           # Root Gradle config
└── settings.gradle        # Includes :android module
```

## Steps

### 1. Compile Native Libraries

From `cmd/android/build/` directory:

```bash
androidcompile.bat
```

This compiles Go code to `.so` libraries for 4 architectures:
- `armeabi-v7a`
- `arm64-v8a`
- `x86`
- `x86_64`

The output goes to `libs/` directory.

### 2. Build the APK

```bash
gradlew.bat assembleDebug
```

Or for release:
```bash
gradlew.bat bundleRelease
```

The APK will be at: `android/build/outputs/apk/debug/app-debug.apk`

### 3. Install to Device

```bash
adb install android/build/outputs/apk/debug/app-debug.apk
```

## Multiplayer (Wi-Fi) on Android

### Requirements

- **Network permissions** - Already added to `AndroidManifest.xml`:
  ```xml
  <uses-permission android:name="android.permission.INTERNET" />
  <uses-permission android:name="android.permission.ACCESS_WIFI_STATE" />
  <uses-permission android:name="android.permission.ACCESS_NETWORK_STATE" />
  ```

- **Same Wi-Fi network** - Host and client must be on the same network.

### How to Play

1. **Start host (Desktop)** - Run desktop version, click "Host Game (Wi-Fi)".
   - The host automatically broadcasts its presence on the local network (UDP port 9001).
2. **Join from Android** - Open the app, click "Join Game (Wi-Fi)".
3. **Select Host** - The app tries to auto-discover hosts. If none found:
   - Click **"Scan TCP"** to scan the local network (fallback for Android)
   - Or click **"Manual IP"** to type the host's IP directly
4. **Connect** - Click on a discovered host or connect with manual IP.
5. **Result** - You'll see all players (host + clients) with distinct colors on both devices.

## Tips & Troubleshooting

- **Cannot connect?** 
  - Verify that port **9000** is not blocked by firewall on the host.
  - On Windows host: Allow `Legiao.exe` through Windows Firewall (inbound rule for port 9000).
  - Ensure host and client are on the same Wi-Fi network.
- **App crashes on start?** Check `logcat` for errors:
  ```bash
  adb logcat | findstr legiao
  ```
- **No internet?** Verify permissions in `AndroidManifest.xml` and grant them in Android Settings → Apps → Legiao → Permissions.
- **Build fails?** Ensure `ANDROID_HOME` and `ANDROID_NDK_HOME` are set correctly.
- **Logs** - On Android, use `adb logcat` to see game logs.

## Development Notes

- The Android version uses `rl.InitWindow(0, 0, ...)` for fullscreen.
- The game logic is shared via `internal/game/loop.go` - same code for Desktop and Android.
- Networking works the same way on both platforms using TCP.
- Player colors are assigned automatically from `entity.PresetColors`.
