@echo off
setlocal

if "%~1"=="" (
  echo 用法：%~n0 "https://music.163.com/#/playlist?id=123456" "D:\Music"
  echo.
  echo 首次使用请先运行：
  echo   setup_netease_batch_windows.bat
  exit /b 1
)

set "URL=%~1"
set "OUTDIR=%~2"
if "%OUTDIR%"=="" set "OUTDIR=%~dp0downloads"

cd /d "%~dp0"

if not exist "config.ini" (
  echo 缺少 config.ini，请先运行 setup_netease_batch_windows.bat。
  exit /b 1
)

if not exist "netease-batch.exe" (
  call "%~dp0build_netease_batch_windows.bat"
  if errorlevel 1 exit /b %errorlevel%
)

"%~dp0netease-batch.exe" -config "%~dp0config.ini" -url "%URL%" -out "%OUTDIR%" -quality lossless -concurrency 4
