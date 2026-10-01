# Build Commands

Run commands from:

```text
cmd/android/build
```

Native libraries, required for debug and release:

```powershell
.\androidcompile.bat
```

Debug APK:

```powershell
.\gradlew.bat assembleDebug
Get-ChildItem android\build\outputs\apk\debug\*.apk | Select-Object FullName,Length,LastWriteTime
Get-FileHash android\build\outputs\apk\debug\*.apk -Algorithm SHA256
```

Release AAB:

```powershell
.\gradlew.bat bundleRelease
Get-ChildItem android\build\outputs\bundle\release\*.aab | Select-Object FullName,Length,LastWriteTime
Get-FileHash android\build\outputs\bundle\release\android-release.aab -Algorithm SHA256
```

Credential presence checks for release:

```powershell
if ($env:KEYSTORE_PASSWORD) { 'KEYSTORE_PASSWORD=set' } else { 'KEYSTORE_PASSWORD=missing' }
if ($env:KEY_PASSWORD) { 'KEY_PASSWORD=set' } else { 'KEY_PASSWORD=missing' }
```

Do not use `full_build.bat` or `run_build.bat` in unattended agent execution because they pause for keyboard input.
