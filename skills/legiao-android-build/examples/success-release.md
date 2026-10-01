# Example: Successful Release AAB

User request:

```text
Generate an Android release AAB.
```

Activation:

```text
Using skill: legiao-android-build
Build type: release
```

Expected steps:

1. Read `AGENTS.md`, `AGENT.md`, and required docs.
2. Inspect `cmd/android/build/android/build.gradle`.
3. Increment `versionCode` and `versionName`.
4. Update `doc/changelog.md`.
5. Check that `KEYSTORE_PASSWORD` and `KEY_PASSWORD` are set without printing values.
6. Run `.\androidcompile.bat`.
7. Run `.\gradlew.bat bundleRelease`.
8. Verify `android/build/outputs/bundle/release/android-release.aab`.
9. Report path, size, SHA-256, and versions used.

Expected log fragments:

```text
compiling for platform armeabi-v7a and architecture arm
compiling for platform arm64-v8a and architecture arm64
compiling for platform x86 and architecture 386
compiling for platform x86_64 and architecture amd64
BUILD SUCCESSFUL
```
