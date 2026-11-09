# Example: Release Failure

User request:

```text
Generate a signed release AAB.
```

Observed issue:

```text
KEYSTORE_PASSWORD=missing
KEY_PASSWORD=set
```

Expected behavior:

- Do not print credential values.
- Do not attempt to bypass release signing without explicit user instruction.
- Report that the release build cannot be completed until `KEYSTORE_PASSWORD` is set.
- Offer to build a debug APK only if that would satisfy the user.

Another common failure:

```text
Downloading https://services.gradle.org/distributions/gradle-8.9-all.zip
PKIX path building failed
```

Expected behavior:

- Identify it as a Gradle/JDK certificate or cache/network issue.
- Retry once if the environment allows normal network/cache access.
- If it still fails, report the failure with the relevant log fragment and do not claim an artifact was produced.
