@echo off
setlocal

where go >nul 2>nul
if errorlevel 1 (
  echo 未找到 Go。请先安装 Go 1.26+：https://go.dev/dl/
  exit /b 1
)

cd /d "%~dp0"
go build -o netease-batch.exe ./cmd/netease-batch
if errorlevel 1 (
  echo 构建 netease-batch.exe 失败。
  exit /b %errorlevel%
)

go build -ldflags "-H=windowsgui" -o setup_netease_batch_windows.exe ./cmd/netease-batch-launcher
if errorlevel 1 (
  echo 构建 setup_netease_batch_windows.exe 失败。
  exit /b %errorlevel%
)

echo 已构建：%cd%\netease-batch.exe
echo 已构建：%cd%\setup_netease_batch_windows.exe
echo.
echo 首次使用：
echo   setup_netease_batch_windows.exe
