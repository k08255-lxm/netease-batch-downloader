package netease

import (
	"fmt"
	"strings"
)

type BrowserCookieResult struct {
	Browser string
	Profile string
	Cookie  string
	MusicU  string
}

func ReadBrowserCookie(browserHint, profileHint string) (*BrowserCookieResult, error) {
	result, err := readBrowserCookie(strings.TrimSpace(browserHint), strings.TrimSpace(profileHint))
	if err != nil {
		return nil, err
	}
	if result == nil || strings.TrimSpace(result.Cookie) == "" || strings.TrimSpace(result.MusicU) == "" {
		return nil, fmt.Errorf("browser cookie missing MUSIC_U")
	}
	return result, nil
}
