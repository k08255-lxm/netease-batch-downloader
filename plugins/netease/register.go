package netease

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/k08255-lxm/netease-batch-downloader/bot/config"
	logpkg "github.com/k08255-lxm/netease-batch-downloader/bot/logger"
	platformplugins "github.com/k08255-lxm/netease-batch-downloader/bot/platform/plugins"
)

func init() {
	if err := platformplugins.Register("netease", buildContribution); err != nil {
		panic(err)
	}
}

func buildContribution(cfg *config.Config, logger *logpkg.Logger) (*platformplugins.Contribution, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config required")
	}
	rawCookie := strings.TrimSpace(strings.Trim(cfg.GetPluginString("netease", "cookie"), "`\"'"))
	musicU := strings.TrimSpace(strings.Trim(cfg.GetPluginString("netease", "music_u"), "`\"'"))
	if musicU == "" {
		musicU = strings.TrimSpace(strings.Trim(cfg.GetString("MUSIC_U"), "`\"'"))
	}
	browser := strings.TrimSpace(cfg.GetPluginString("netease", "browser"))
	browserProfile := strings.TrimSpace(cfg.GetPluginString("netease", "browser_profile"))
	spoofIP := true
	if pluginCfg, ok := cfg.GetPluginConfig("netease"); ok {
		if _, exists := pluginCfg["spoof_ip"]; exists {
			spoofIP = cfg.GetPluginBool("netease", "spoof_ip")
		}
	}
	autoRenewEnabled := cfg.GetPluginBool("netease", "auto_renew_enabled")
	intervalSec := cfg.GetPluginInt("netease", "auto_renew_interval_sec")
	var interval time.Duration
	if intervalSec > 0 {
		interval = time.Duration(intervalSec) * time.Second
	}
	persist := func(pairs map[string]string) error {
		return cfg.PersistPluginConfig("netease", pairs)
	}
	client := New("", spoofIP, nil, persist)
	client.logger = logger
	switch {
	case rawCookie != "":
		if err := client.LoadCookieString(rawCookie); err != nil {
			return nil, fmt.Errorf("load netease cookie from config: %w", err)
		}
	case musicU != "":
		client = New(musicU, spoofIP, logger, persist)
	default:
		result, err := ReadBrowserCookie(browser, browserProfile)
		if err != nil {
			if browser != "" && !strings.EqualFold(browser, "auto") {
				return nil, fmt.Errorf("load netease cookie from browser: %w", err)
			}
		} else {
			if err := client.LoadCookieString(result.Cookie); err != nil {
				return nil, fmt.Errorf("load netease cookie from %s/%s: %w", result.Browser, result.Profile, err)
			}
			if logger != nil {
				logger.Info("loaded netease cookie from browser", "browser", result.Browser, "profile", result.Profile)
			}
		}
	}
	client.ConfigureAutoRenew(autoRenewEnabled, interval)
	client.StartAutoRenewDaemon(context.Background())
	if err := client.SetAPIProxy(cfg.ResolveAPIProxyConfig("netease")); err != nil {
		return nil, err
	}
	disableRadar := true
	if pluginCfg, ok := cfg.GetPluginConfig("netease"); ok {
		if _, exists := pluginCfg["disable_radar"]; exists {
			disableRadar = cfg.GetPluginBool("netease", "disable_radar")
		}
	}
	platform := NewPlatform(client, disableRadar)
	id3Provider := NewID3Provider(client)

	contrib := &platformplugins.Contribution{
		Platform: platform,
		ID3:      id3Provider,
	}

	if cfg.GetBool("EnableRecognize") {
		recognizeService := NewRecognizeService(cfg.GetInt("RecognizePort"))
		contrib.Recognizer = NewRecognizer(recognizeService)
	}

	return contrib, nil
}
