# Example: Successful Debug APK

User request:

```text
Build a debug APK for testing.
```

Activation:

```text
Using skill: legiao-android-build
Build type: debug
```

Expected steps:

1. Read `AGENTS.md`, `AGENT.md`, and required docs.
2. Do not edit `versionCode` or `versionName`.
3. Run `.\androidcompile.bat`.
4. Run `.\gradlew.bat assembleDebug`.
5. Verify an APK under `android/build/outputs/apk/debug/`.
6. Report path, size, SHA-256, and that app version was not changed.

Expected log fragments:

```text
compiling for platform arm64-v8a and architecture arm64
BUILD SUCCESSFUL
```
