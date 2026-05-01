# Implementation Plan: Adding Android Support

## Context
The goal is to enable Android support for the project while maintaining a clean, working Windows desktop build. Currently, the project uses `raylib-go` and is structured with `cmd/desktop` as the primary entry point. Android needs a separate entry point and specific build infrastructure (Gradle/NDK).

## Implementation Approach
1.  **Platform Abstraction**: Ensure `internal/` packages remain platform-agnostic. Use Go build tags (`//go:build android`) in any `cmd/` or `internal/platform` files that differ between desktop and mobile (e.g., input polling, resource paths).
2.  **Android Entry Point**: Create `cmd/android/main.go` which uses `rl.SetMain()` to interface with the Android lifecycle.
3.  **Build Infrastructure**: Adapt `cmd/android/build` (the provided template) to include the current project's assets, game modules, and `go.mod`.
4.  **UI/Input**: Utilize the existing `internal/ui/hud.go` (VirtualJoystick) to handle touch input on Android.
5.  **Documentation**: Create `doc/android.md` covering prerequisites (NDK/SDK), environment setup, and compilation commands (`go build`, `gomobile`, or Gradle).

## Critical Files to Modify/Create
- `cmd/android/main.go` (New)
- `cmd/android/build/build.gradle` (Adapt)
- `doc/android.md` (New)

## Verification
- **Windows**: Build and run as usual (`go run cmd/desktop/main.go`).
- **Android**: Use the provided `cmd/android/build` structure to attempt an Android build via `gomobile` or Gradle, ensuring it generates an APK.
