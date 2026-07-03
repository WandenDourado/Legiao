@echo off
cd /d %~dp0

echo ===== STEP 1: Compiling native libraries =====
call androidcompile.bat
if %errorlevel% neq 0 (
    echo FAILED: androidcompile.bat returned error %errorlevel%
    goto :end
)
echo.

echo ===== STEP 2: Verifying .so files =====
dir android\libs\arm64-v8a\liblegiao.so
dir android\libs\armeabi-v7a\liblegiao.so
dir android\libs\x86\liblegiao.so
dir android\libs\x86_64\liblegiao.so
echo.

echo ===== STEP 3: Building AAB =====
call gradlew bundleRelease
if %errorlevel% neq 0 (
    echo FAILED: gradlew bundleRelease returned error %errorlevel%
    goto :end
)
echo.

echo ===== STEP 4: Verifying AAB =====
dir android\build\outputs\bundle\release\*.aab 2>nul
if %errorlevel% neq 0 (
    echo WARNING: No AAB found in expected location
    dir /s /b android\build\outputs\*.aab 2>nul
)

:end
echo.
echo ===== BUILD PROCESS COMPLETE =====
pause
