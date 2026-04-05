//go:build !windows

package netease

import "fmt"

func readBrowserCookie(browserHint, profileHint string) (*BrowserCookieResult, error) {
	_ = browserHint
	_ = profileHint
	return nil, fmt.Errorf("browser cookie auto-read is currently supported on Windows only")
}
