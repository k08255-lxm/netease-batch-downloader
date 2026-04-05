@echo off
setlocal

where go >nul 2>nul
if errorlevel 1 (
  echo Go not found. Install Go 1.26+ first: https://go.dev/dl/
  exit /b 1
)

cd /d "%~dp0"
go build -o netease-batch.exe ./cmd/netease-batch
if errorlevel 1 (
  echo Build failed.
  exit /b %errorlevel%
)

echo Built: %cd%\netease-batch.exe
