# Skill: legiao-android-build

## Name

`legiao-android-build`

## Objective

Build the Legiao Android app in a platform-agnostic way for either:

- Debug APK builds for local testing.
- Release AAB builds for distribution.

The skill preserves the project-specific Android/Raylib build sequence and the release-version rule: every release build must update both `versionCode` and `versionName`; debug builds must not change app version unless explicitly requested.

## Use Cases

Use this skill when:

- The user asks to build Android.
- The user asks for a debug APK.
- The user asks for a release AAB.
- The user asks to run `androidcompile.bat`, `assembleDebug`, or `bundleRelease`.
- The user asks to prepare a signed Android release.
- The user asks to verify Android build artifacts.

## Do Not Use

Do not use this skill when:

- The request is about desktop builds only.
- The request is about code review, gameplay, networking, maps, or UI with no Android artifact requested.
- The user asks only for conceptual documentation and explicitly does not want commands run.
- The repository is not Legiao or does not contain `cmd/android/build/androidcompile.bat`.

## Expected Inputs

The agent should determine or receive:

- Repository root path.
- Build type: `debug` or `release`.
- For release builds, desired `versionCode` and `versionName`, or permission to choose the next sensible values.
- Shell capability for running build commands.
- File edit capability if release version fields must be updated.

If the build type is ambiguous:

- Treat `.aab`, Play Store, signed, production, or `bundleRelease` as release.
- Treat APK, local install, test build, or `assembleDebug` as debug.
- If the user only says "Android build" and artifact type matters, ask whether they want debug APK or release AAB.

## Expected Outputs

Debug output:

- APK under `cmd/android/build/android/build/outputs/apk/debug/`.
- Confirmation that app version was not changed.
- File path, size, and hash when practical.

Release output:

- AAB under `cmd/android/build/android/build/outputs/bundle/release/android-release.aab`.
- Updated `versionCode` and `versionName`.
- Updated `doc/changelog.md`.
- File path, size, and SHA-256.

## Required Context

From the repository root, read:

- `AGENTS.md`
- `AGENT.md`
- `doc/documentation_rules.md`
- `doc/coding_patterns.md`
- `doc/android.md`
- `doc/android_support.md`
- `doc/running_android.md`

If changing structure or adding files, also read:

- `doc/project_structure.md`

## MCP Compatibility

Required MCP servers:

- None.

This skill can run with ordinary filesystem and shell capabilities. If an agent uses MCP, use generic MCP tools only:

- Filesystem read/write contract:
  - Input: repository-relative path and content or patch.
  - Output: success/failure and resulting file metadata.
- Shell command contract:
  - Input: command string and working directory.
  - Output: exit code, stdout, stderr, and duration.
- Git status/diff contract, optional:
  - Input: repository root.
  - Output: changed paths and diff summary.

Do not depend on Codex-only, Claude-only, Devin-only, Cursor-only, or Windsurf-only APIs.

## Execution Flow

### 1. Locate The Repository

Find the directory containing:

- `go.mod`
- `AGENTS.md` or `AGENT.md`
- `cmd/android/build`
- `internal/`

Use `cmd/android/build` as the working directory for build commands.

### 2. Select Build Type

Classify the request as debug or release using the rules in `Expected Inputs`.

Report activation in the working notes or final answer:

```text
Using skill: legiao-android-build
Build type: debug
```

or:

```text
Using skill: legiao-android-build
Build type: release
```

### 3. Prepare Release Version Only For Release

For release builds, edit `cmd/android/build/android/build.gradle` before building.

Default version rule:

- Increment `versionCode` by `1`.
- Increment `versionName` patch by `1`, for example `1.0.8` to `1.0.9`.

If the user supplies exact values, use those values.

After editing version fields:

- Update `doc/changelog.md`.
- Keep the entry concise: date, version bump, modified files, generated artifact.

For debug builds:

- Do not edit `versionCode`.
- Do not edit `versionName`.
- Do not update changelog unless another file was intentionally changed.

### 4. Check Sensitive Environment For Release

For release builds, check only presence:

```powershell
if ($env:KEYSTORE_PASSWORD) { 'KEYSTORE_PASSWORD=set' } else { 'KEYSTORE_PASSWORD=missing' }
if ($env:KEY_PASSWORD) { 'KEY_PASSWORD=set' } else { 'KEY_PASSWORD=missing' }
```

Never print actual credential values.

### 5. Build Native Libraries

Run this before both debug and release Gradle builds:

```powershell
.\androidcompile.bat
```

Expected log pattern:

```text
compiling for platform armeabi-v7a and architecture arm
compiling for platform arm64-v8a and architecture arm64
compiling for platform x86 and architecture 386
compiling for platform x86_64 and architecture amd64
```

Verify libraries:

```powershell
Get-ChildItem android\libs\armeabi-v7a\liblegiao.so,android\libs\arm64-v8a\liblegiao.so,android\libs\x86\liblegiao.so,android\libs\x86_64\liblegiao.so | Select-Object FullName,Length,LastWriteTime
```

### 6. Run Gradle

Debug:

```powershell
.\gradlew.bat assembleDebug
```

Release:

```powershell
.\gradlew.bat bundleRelease
```

Expected success pattern:

```text
BUILD SUCCESSFUL
```

Avoid `full_build.bat` and `run_build.bat` for automated agents because they end with `pause`.

### 7. Verify Artifacts

Debug APK:

```powershell
Get-ChildItem android\build\outputs\apk\debug\*.apk | Select-Object FullName,Length,LastWriteTime
Get-FileHash android\build\outputs\apk\debug\*.apk -Algorithm SHA256
```

Release AAB:

```powershell
Get-ChildItem android\build\outputs\bundle\release\*.aab | Select-Object FullName,Length,LastWriteTime
Get-FileHash android\build\outputs\bundle\release\android-release.aab -Algorithm SHA256
```

## Quality Criteria

The execution is successful only if:

- The correct build type was selected.
- Required project docs were read.
- Native libraries were generated or confirmed for all four ABIs.
- Gradle completed with `BUILD SUCCESSFUL`.
- The expected APK or AAB exists.
- Release builds updated both `versionCode` and `versionName`.
- Debug builds did not change version fields.
- Release builds did not expose signing secrets.
- The final report includes artifact path, size, hash, and relevant version values.

## Observability

Skill activation is observable when the agent reports:

```text
Using skill: legiao-android-build
Build type: <debug|release>
```

Correct execution is observable through:

- Native ABI compile lines from `androidcompile.bat`.
- `.so` files under `cmd/android/build/android/libs/`.
- Gradle `BUILD SUCCESSFUL`.
- APK or AAB output path.
- Hash output from `Get-FileHash` or equivalent.

Debugging failure:

- If native compilation fails with missing NDK, inspect `androidcompile.bat` SDK/NDK paths.
- If Gradle cannot download or resolve dependencies, inspect network/certificate/cache errors.
- If release signing fails, check only whether `KEYSTORE_PASSWORD` and `KEY_PASSWORD` are set.
- If AAB is missing after success, search under `cmd/android/build/android/build/outputs/`.

## Security Rules

- Do not execute destructive cleanup commands without explicit confirmation.
- Do not modify files outside the repository scope.
- Do not print or persist secret values.
- Do not assume admin privileges.
- Do not change debug version fields unless explicitly requested.
- Do not commit, push, upload, or publish artifacts unless explicitly requested.

## Complete Examples

### Success: Release AAB

User request:

```text
Generate a release AAB for Android.
```

Expected agent behavior:

```text
Using skill: legiao-android-build
Build type: release
Read AGENTS.md, AGENT.md, documentation_rules.md, coding_patterns.md, android docs.
Increment versionCode 10 -> 11 and versionName 1.0.8 -> 1.0.9.
Update doc/changelog.md.
Run .\androidcompile.bat.
Run .\gradlew.bat bundleRelease.
Verify android-release.aab, size, and SHA-256.
```

### Success: Debug APK

User request:

```text
Build a debug APK for local testing.
```

Expected agent behavior:

```text
Using skill: legiao-android-build
Build type: debug
Read required docs.
Do not change versionCode/versionName.
Run .\androidcompile.bat.
Run .\gradlew.bat assembleDebug.
Verify APK path and hash.
```

### Failure: Missing Signing Environment

Observed output:

```text
KEYSTORE_PASSWORD=missing
KEY_PASSWORD=set
```

Expected agent behavior:

```text
Stop before release signing or report that release cannot be completed until KEYSTORE_PASSWORD is set.
Do not print secret values.
Do not attempt to bypass signing unless the user explicitly requests an unsigned/debug artifact.
```

### Incorrect Use

User request:

```text
Review the enemy spawn logic.
```

Expected agent behavior:

```text
Do not use legiao-android-build.
Use gameplay/system documentation instead.
```

### Skill Should Not Be Triggered

User request:

```text
Explain how the camera follows the player.
```

Expected agent behavior:

```text
Do not use legiao-android-build.
Read camera documentation and relevant game files.
```

## Self-Evaluation

At the end of execution, answer:

- Was `legiao-android-build` appropriate for this context?
- Which build type was selected and why?
- Which quality criteria were applied?
- Which tools were used?
- What is the confidence level in the result?
- Are there relevant limitations, such as missing SDK, missing credentials, or unverified device install?
