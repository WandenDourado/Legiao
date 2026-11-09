# Example: Incorrect Use And Non-Trigger

Incorrect use:

```text
Review the enemy spawn logic.
```

Do not activate `legiao-android-build`. The task touches gameplay systems, not Android build artifacts.

Non-trigger:

```text
Explain how camera clamping works.
```

Do not activate `legiao-android-build`. Read `doc/camera.md` and relevant game files instead.

Ambiguous trigger:

```text
Build Android.
```

If the user did not specify APK, AAB, debug, or release, ask whether they want a debug APK or release AAB. If the environment or user context strongly implies AAB/release, state the assumption before proceeding.
