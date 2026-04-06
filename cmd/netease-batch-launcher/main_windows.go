//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"
)

const (
	mbOK          = 0x00000000
	mbIconError   = 0x00000010
	mbIconWarning = 0x00000030
)

func main() {
	exePath, err := os.Executable()
	if err != nil {
		showMessage("启动失败", fmt.Sprintf("无法确定程序路径：%v", err), mbIconError)
		os.Exit(1)
	}
	baseDir := filepath.Dir(exePath)
	scriptPath := filepath.Join(baseDir, "setup_netease_batch_windows.ps1")
	if _, err := os.Stat(scriptPath); err != nil {
		showMessage("缺少文件", fmt.Sprintf("未找到启动脚本：\n%s", scriptPath), mbIconError)
		os.Exit(1)
	}

	shellPath, err := findPowerShell()
	if err != nil {
		showMessage("缺少 PowerShell", err.Error(), mbIconWarning)
		os.Exit(1)
	}

	args := []string{
		"-NoProfile",
		"-ExecutionPolicy", "Bypass",
		"-File", scriptPath,
	}
	if len(os.Args) > 1 {
		args = append(args, os.Args[1:]...)
	}

	cmd := exec.Command(shellPath, args...)
	cmd.Dir = baseDir
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	if err := cmd.Run(); err != nil {
		showMessage("启动失败", fmt.Sprintf("启动 Windows 向导失败：%v", err), mbIconError)
		os.Exit(1)
	}
}

func findPowerShell() (string, error) {
	for _, candidate := range []string{"pwsh.exe", "powershell.exe"} {
		if path, err := exec.LookPath(candidate); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("未找到 PowerShell。请安装 PowerShell 7，或启用系统自带 Windows PowerShell 后再试。")
}

func showMessage(title, text string, icon uintptr) {
	user32 := syscall.NewLazyDLL("user32.dll")
	messageBox := user32.NewProc("MessageBoxW")
	titlePtr, _ := syscall.UTF16PtrFromString(title)
	textPtr, _ := syscall.UTF16PtrFromString(text)
	_, _, _ = messageBox.Call(
		0,
		uintptr(unsafe.Pointer(textPtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		mbOK|icon,
	)
}
