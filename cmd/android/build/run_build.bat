@echo off
cd /d %~dp0
echo === Running androidcompile.bat ===
call androidcompile.bat
echo.
echo === Checking .so files ===
dir android\libs\arm64-v8a\
dir android\libs\armeabi-v7a\
dir android\libs\x86\
dir android\libs\x86_64\
echo.
echo === Running gradlew bundleRelease ===
call gradlew bundleRelease
echo.
echo === Checking AAB output ===
dir android\build\outputs\bundle\release\
echo.
echo DONE. Press any key to exit.
pause
