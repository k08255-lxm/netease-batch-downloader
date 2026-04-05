//go:build windows

package netease

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"unsafe"

	_ "modernc.org/sqlite"
)

var errBrowserCookieNotFound = errors.New("netease browser cookie not found")

type windowsBrowserSpec struct {
	Name        string
	UserDataDir string
	LocalState  string
	ProfilesDir string
	Firefox     bool
}

type chromiumLocalState struct {
	OSCrypt struct {
		EncryptedKey string `json:"encrypted_key"`
	} `json:"os_crypt"`
}

type dataBlob struct {
	cbData uint32
	pbData *byte
}

func readBrowserCookie(browserHint, profileHint string) (*BrowserCookieResult, error) {
	browsers, err := candidateWindowsBrowsers(browserHint)
	if err != nil {
		return nil, err
	}

	var errs []error
	for _, browser := range browsers {
		result, readErr := browser.read(profileHint)
		if readErr == nil {
			return result, nil
		}
		errs = append(errs, fmt.Errorf("%s: %w", browser.Name, readErr))
	}
	if len(errs) == 0 {
		return nil, errBrowserCookieNotFound
	}
	return nil, errors.Join(errs...)
}

func candidateWindowsBrowsers(browserHint string) ([]windowsBrowserSpec, error) {
	localAppData := strings.TrimSpace(os.Getenv("LOCALAPPDATA"))
	appData := strings.TrimSpace(os.Getenv("APPDATA"))

	all := []windowsBrowserSpec{
		{
			Name:        "edge",
			UserDataDir: filepath.Join(localAppData, "Microsoft", "Edge", "User Data"),
			LocalState:  filepath.Join(localAppData, "Microsoft", "Edge", "User Data", "Local State"),
		},
		{
			Name:        "chrome",
			UserDataDir: filepath.Join(localAppData, "Google", "Chrome", "User Data"),
			LocalState:  filepath.Join(localAppData, "Google", "Chrome", "User Data", "Local State"),
		},
		{
			Name:        "brave",
			UserDataDir: filepath.Join(localAppData, "BraveSoftware", "Brave-Browser", "User Data"),
			LocalState:  filepath.Join(localAppData, "BraveSoftware", "Brave-Browser", "User Data", "Local State"),
		},
		{
			Name:        "firefox",
			ProfilesDir: filepath.Join(appData, "Mozilla", "Firefox", "Profiles"),
			Firefox:     true,
		},
	}

	if browserHint == "" || strings.EqualFold(browserHint, "auto") {
		return all, nil
	}

	normalized := normalizeBrowserName(browserHint)
	for _, browser := range all {
		if browser.Name == normalized {
			return []windowsBrowserSpec{browser}, nil
		}
	}
	return nil, fmt.Errorf("unsupported browser %q", browserHint)
}

func normalizeBrowserName(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "edge", "msedge", "microsoft-edge":
		return "edge"
	case "chrome", "google-chrome":
		return "chrome"
	case "brave", "brave-browser":
		return "brave"
	case "firefox", "ff":
		return "firefox"
	default:
		return strings.ToLower(strings.TrimSpace(name))
	}
}

func (b windowsBrowserSpec) read(profileHint string) (*BrowserCookieResult, error) {
	if b.Firefox {
		return b.readFirefox(profileHint)
	}
	return b.readChromium(profileHint)
}

func (b windowsBrowserSpec) readChromium(profileHint string) (*BrowserCookieResult, error) {
	if strings.TrimSpace(b.UserDataDir) == "" || strings.TrimSpace(b.LocalState) == "" {
		return nil, errBrowserCookieNotFound
	}
	if _, err := os.Stat(b.UserDataDir); err != nil {
		return nil, errBrowserCookieNotFound
	}
	key, err := readChromiumKey(b.LocalState)
	if err != nil {
		return nil, err
	}

	profiles, err := chromiumProfiles(b.UserDataDir, profileHint)
	if err != nil {
		return nil, err
	}
	for _, profile := range profiles {
		cookiePath := chromiumCookiePath(b.UserDataDir, profile)
		cookies, readErr := readChromiumCookies(cookiePath, key)
		if readErr != nil {
			continue
		}
		musicU := musicUFromCookies(cookies)
		if musicU == "" {
			continue
		}
		return &BrowserCookieResult{
			Browser: b.Name,
			Profile: profile,
			Cookie:  renderNeteaseCookie(cookies),
			MusicU:  musicU,
		}, nil
	}
	return nil, errBrowserCookieNotFound
}

func (b windowsBrowserSpec) readFirefox(profileHint string) (*BrowserCookieResult, error) {
	if strings.TrimSpace(b.ProfilesDir) == "" {
		return nil, errBrowserCookieNotFound
	}
	if _, err := os.Stat(b.ProfilesDir); err != nil {
		return nil, errBrowserCookieNotFound
	}

	profiles, err := firefoxProfiles(b.ProfilesDir, profileHint)
	if err != nil {
		return nil, err
	}
	for _, profile := range profiles {
		cookiePath := filepath.Join(b.ProfilesDir, profile, "cookies.sqlite")
		cookies, readErr := readFirefoxCookies(cookiePath)
		if readErr != nil {
			continue
		}
		musicU := musicUFromCookies(cookies)
		if musicU == "" {
			continue
		}
		return &BrowserCookieResult{
			Browser: b.Name,
			Profile: profile,
			Cookie:  renderNeteaseCookie(cookies),
			MusicU:  musicU,
		}, nil
	}
	return nil, errBrowserCookieNotFound
}

func chromiumProfiles(userDataDir, profileHint string) ([]string, error) {
	if trimmed := strings.TrimSpace(profileHint); trimmed != "" {
		return []string{trimmed}, nil
	}
	entries, err := os.ReadDir(userDataDir)
	if err != nil {
		return nil, err
	}
	profiles := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "Default" || strings.HasPrefix(name, "Profile ") {
			profiles = append(profiles, name)
		}
	}
	sort.Strings(profiles)
	return profiles, nil
}

func firefoxProfiles(profilesDir, profileHint string) ([]string, error) {
	if trimmed := strings.TrimSpace(profileHint); trimmed != "" {
		return []string{trimmed}, nil
	}
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		return nil, err
	}
	profiles := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			profiles = append(profiles, entry.Name())
		}
	}
	sort.Strings(profiles)
	return profiles, nil
}

func chromiumCookiePath(userDataDir, profile string) string {
	networkPath := filepath.Join(userDataDir, profile, "Network", "Cookies")
	if _, err := os.Stat(networkPath); err == nil {
		return networkPath
	}
	return filepath.Join(userDataDir, profile, "Cookies")
}

func readChromiumKey(localStatePath string) ([]byte, error) {
	data, err := os.ReadFile(localStatePath)
	if err != nil {
		return nil, err
	}
	var state chromiumLocalState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	encoded := strings.TrimSpace(state.OSCrypt.EncryptedKey)
	if encoded == "" {
		return nil, fmt.Errorf("chromium local state missing encrypted key")
	}
	encryptedKey, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	encryptedKey = bytes.TrimPrefix(encryptedKey, []byte("DPAPI"))
	return decryptDPAPI(encryptedKey)
}

func readChromiumCookies(cookiePath string, key []byte) ([]*http.Cookie, error) {
	dbPath, cleanup, err := cloneSQLiteForRead(cookiePath)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT name, value, encrypted_value
		FROM cookies
		WHERE host_key IN (?, ?)
		ORDER BY last_access_utc DESC, creation_utc DESC
	`, "music.163.com", ".music.163.com")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seen := make(map[string]struct{})
	cookies := make([]*http.Cookie, 0, 8)
	for rows.Next() {
		var name string
		var value string
		var encrypted []byte
		if err := rows.Scan(&name, &value, &encrypted); err != nil {
			return nil, err
		}
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		normalizedName := strings.ToUpper(name)
		if _, exists := seen[normalizedName]; exists {
			continue
		}
		if strings.TrimSpace(value) == "" {
			value, err = decryptChromiumCookieValue(encrypted, key)
			if err != nil {
				continue
			}
		}
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		cookies = append(cookies, &http.Cookie{Name: name, Value: value})
		seen[normalizedName] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(cookies) == 0 {
		return nil, errBrowserCookieNotFound
	}
	sortCookiesByName(cookies)
	return cookies, nil
}

func readFirefoxCookies(cookiePath string) ([]*http.Cookie, error) {
	dbPath, cleanup, err := cloneSQLiteForRead(cookiePath)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT name, value
		FROM moz_cookies
		WHERE host IN (?, ?)
		ORDER BY lastAccessed DESC, creationTime DESC
	`, "music.163.com", ".music.163.com")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seen := make(map[string]struct{})
	cookies := make([]*http.Cookie, 0, 8)
	for rows.Next() {
		var name string
		var value string
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		if name == "" || value == "" {
			continue
		}
		normalizedName := strings.ToUpper(name)
		if _, exists := seen[normalizedName]; exists {
			continue
		}
		cookies = append(cookies, &http.Cookie{Name: name, Value: value})
		seen[normalizedName] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(cookies) == 0 {
		return nil, errBrowserCookieNotFound
	}
	sortCookiesByName(cookies)
	return cookies, nil
}

func cloneSQLiteForRead(path string) (string, func(), error) {
	if _, err := os.Stat(path); err != nil {
		return "", func() {}, err
	}
	source, err := os.Open(path)
	if err != nil {
		return "", func() {}, err
	}
	defer source.Close()

	tempFile, err := os.CreateTemp("", "netease-browser-cookie-*.sqlite")
	if err != nil {
		return "", func() {}, err
	}
	tempPath := tempFile.Name()
	cleanup := func() {
		_ = os.Remove(tempPath)
	}
	if _, err := io.Copy(tempFile, source); err != nil {
		tempFile.Close()
		cleanup()
		return "", func() {}, err
	}
	if err := tempFile.Close(); err != nil {
		cleanup()
		return "", func() {}, err
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		sourceSidecar := path + suffix
		if _, err := os.Stat(sourceSidecar); err != nil {
			continue
		}
		if err := copyFile(sourceSidecar, tempPath+suffix); err != nil {
			cleanup()
			_ = os.Remove(tempPath + "-wal")
			_ = os.Remove(tempPath + "-shm")
			return "", func() {}, err
		}
	}
	cleanup = func() {
		_ = os.Remove(tempPath)
		_ = os.Remove(tempPath + "-wal")
		_ = os.Remove(tempPath + "-shm")
	}
	return tempPath, cleanup, nil
}

func copyFile(sourcePath, destPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()

	dest, err := os.Create(destPath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dest, source); err != nil {
		dest.Close()
		return err
	}
	return dest.Close()
}

func decryptChromiumCookieValue(encryptedValue, key []byte) (string, error) {
	if len(encryptedValue) == 0 {
		return "", fmt.Errorf("empty encrypted cookie")
	}
	if bytes.HasPrefix(encryptedValue, []byte("v10")) || bytes.HasPrefix(encryptedValue, []byte("v11")) {
		payload := encryptedValue[3:]
		if len(payload) < 12+16 {
			return "", fmt.Errorf("invalid chromium aes-gcm payload")
		}
		block, err := aes.NewCipher(key)
		if err != nil {
			return "", err
		}
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return "", err
		}
		nonce := payload[:12]
		ciphertext := payload[12:]
		plain, err := gcm.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			return "", err
		}
		return string(plain), nil
	}
	plain, err := decryptDPAPI(encryptedValue)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func decryptDPAPI(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty dpapi input")
	}
	crypt32 := syscall.NewLazyDLL("Crypt32.dll")
	kernel32 := syscall.NewLazyDLL("Kernel32.dll")
	procCryptUnprotectData := crypt32.NewProc("CryptUnprotectData")
	procLocalFree := kernel32.NewProc("LocalFree")

	in := dataBlob{
		cbData: uint32(len(data)),
		pbData: &data[0],
	}
	var out dataBlob
	ret, _, callErr := procCryptUnprotectData.Call(
		uintptr(unsafe.Pointer(&in)),
		0,
		0,
		0,
		0,
		0,
		uintptr(unsafe.Pointer(&out)),
	)
	if ret == 0 {
		if callErr != syscall.Errno(0) {
			return nil, callErr
		}
		return nil, fmt.Errorf("CryptUnprotectData failed")
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(out.pbData)))

	size := int(out.cbData)
	plain := make([]byte, size)
	copy(plain, unsafe.Slice(out.pbData, size))
	return plain, nil
}

func musicUFromCookies(cookies []*http.Cookie) string {
	for _, cookie := range cookies {
		if cookie == nil || !strings.EqualFold(strings.TrimSpace(cookie.Name), "MUSIC_U") {
			continue
		}
		return strings.TrimSpace(cookie.Value)
	}
	return ""
}

func sortCookiesByName(cookies []*http.Cookie) {
	sort.Slice(cookies, func(i, j int) bool {
		left := strings.ToLower(strings.TrimSpace(cookies[i].Name))
		right := strings.ToLower(strings.TrimSpace(cookies[j].Name))
		return left < right
	})
}
