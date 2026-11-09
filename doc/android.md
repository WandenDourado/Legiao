# Android

Use this doc for Android build, APK packaging, manifest, NDK, and APK asset access. For actual artifact builds, use the `legiao-android-build` skill.

## Prerequisites

- Go in `PATH`.
- `ANDROID_HOME` set to the Android SDK.
- `ANDROID_NDK_HOME` set to the Android NDK used by the build scripts.
- Gradle wrapper in `cmd/android/build/`.

## Build Commands

From `cmd/android/build/`:

```cmd
.\androidcompile.bat
.\gradlew.bat assembleDebug
```

> These are Windows batch files — they run **only under `cmd.exe`**, not in a
> bash/PowerShell wrapper that issues `cmd /c "call ..."` (that path is
> unreliable here). From PowerShell, invoke `cmd /c` explicitly:
> `& cmd /c '.\gradlew.bat assembleDebug'`.

> **Trap — corrupted `gradle-wrapper.jar`:** the checked-in
> `gradle/wrapper/gradle-wrapper.jar` can lose its manifest (git may have
> normalized the binary), failing with `no main manifest attribute, in
> gradle\wrapper\gradle-wrapper.jar` and no other output. When `gradlew.bat`
> silently does nothing, run Gradle **directly** from the cached distribution:
>
> ```powershell
> $g = Get-ChildItem "$env:USERPROFILE\.gradle\wrapper\dists\gradle-*-all\*\gradle-*\bin\gradle.bat" | Select-Object -First 1
> & cmd /c "$g assembleDebug"
> ```
>
> This repo pins Gradle 8.9. Requires JDK 17+ (JDK 21 at
> `C:\Program Files\Java\jdk-21`); the JDK 8 on PATH will fail — ensure
> `JAVA_HOME` points at a valid JDK.

Debug APK output:

```text
cmd/android/build/android/build/outputs/apk/debug/android-debug.apk
```

Release bundle, when signing env vars are configured:

```powershell
.\gradlew.bat bundleRelease
```

## Runtime Assets

All runtime files needed on Android must live under project-root `assets/`. The build script copies this tree into:

```text
cmd/android/build/android/assets/
```

Rules:

- Use `assets.Path()` for every `rl.Load*` path.
- Use `internal/tilemap/readFile()` wrappers for map/TSX data; Android uses `rl.OpenAsset()`.
- Do not use `os.ReadFile` directly for runtime game data that must work inside the APK.
- Maps used at runtime should be loaded from `assets/maps/...`.
- Tileset references inside maps should stay relative within the `assets/` tree, for example `../tilesets/title_set.tsx`.

## Manifest And Network

The Android manifest must include local-network permissions used by multiplayer discovery/connect:

```xml
<uses-permission android:name="android.permission.INTERNET" />
<uses-permission android:name="android.permission.ACCESS_WIFI_STATE" />
<uses-permission android:name="android.permission.ACCESS_NETWORK_STATE" />
<uses-permission android:name="android.permission.CHANGE_WIFI_STATE" />
<uses-permission android:name="android.permission.CHANGE_WIFI_MULTICAST_STATE" />
```

The app is Go/raylib driven and should not require custom Java application code for discovery.

## Install And Logs

```powershell
adb install android/build/outputs/apk/debug/android-debug.apk
adb logcat | findstr Legiao
```

If the app exits on launch, first check asset paths and Android asset reading. Most launch failures after map changes come from missing APK assets or direct filesystem reads.
