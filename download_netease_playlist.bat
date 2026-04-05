@echo off
setlocal

if "%~1"=="" (
  echo Usage: %~n0 "https://music.163.com/#/playlist?id=123456" "D:\Music"
  echo.
  echo First run:
  echo 1. Copy config_example.ini to config.ini
  echo 2. Fill [plugins.netease] music_u
  echo 3. Run this script again
  exit /b 1
)

set "URL=%~1"
set "OUTDIR=%~2"
if "%OUTDIR%"=="" set "OUTDIR=%~dp0downloads"

cd /d "%~dp0"

if not exist "config.ini" (
  echo Missing config.ini. Copy config_example.ini to config.ini and fill [plugins.netease] music_u first.
  exit /b 1
)

if not exist "netease-batch.exe" (
  call "%~dp0build_netease_batch_windows.bat"
  if errorlevel 1 exit /b %errorlevel%
)

"%~dp0netease-batch.exe" -config "%~dp0config.ini" -url "%URL%" -out "%OUTDIR%" -quality lossless -concurrency 4
