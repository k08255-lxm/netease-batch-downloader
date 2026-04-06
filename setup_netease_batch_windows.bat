@echo off
setlocal
cd /d "%~dp0"

where pwsh >nul 2>nul
if not errorlevel 1 (
  pwsh -NoProfile -ExecutionPolicy Bypass -File "%~dp0setup_netease_batch_windows.ps1"
) else (
  powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0setup_netease_batch_windows.ps1"
)
