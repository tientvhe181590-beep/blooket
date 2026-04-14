@echo off
setlocal EnableExtensions
cd /d "%~dp0"

echo.
echo ========================================
echo  Blooket CSV Exporter - Windows build
echo ========================================
echo.

set "CGO_ENABLED=1"
echo [1/2] CGO_ENABLED=1 ^(required for Fyne^)
echo [2/2] go build ...
echo.

go build -trimpath -ldflags="-H windowsgui -s -w" -o blooket-csv-exporter.exe .
set "EXITCODE=%ERRORLEVEL%"

echo.
if %EXITCODE% neq 0 (
  echo *** BUILD FAILED - exit code %EXITCODE% ***
) else (
  echo *** BUILD OK ***
  echo Output: %CD%\blooket-csv-exporter.exe
)
echo.

if /i not "%~1"=="nopause" (
  pause
)

endlocal & exit /b %EXITCODE%
