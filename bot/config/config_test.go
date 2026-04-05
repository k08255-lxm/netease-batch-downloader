package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadINI(t *testing.T) {
	path := filepath.Join("..", "..", "config_example.ini")
	conf, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if conf.GetInt("DownloadTimeout") != 60 {
		t.Fatalf("expected default DownloadTimeout=60, got %d", conf.GetInt("DownloadTimeout"))
	}

	if !conf.GetBool("CheckMD5") {
		t.Fatalf("expected CheckMD5 default to be true")
	}
}

func TestPluginSections(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_config_*.ini")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	configContent := `MUSIC_U = test_music_u

[plugins.netease]
api_url = https://netease.api
retry = 3
feature_enabled = true

[plugins.custom]
client_id = custom_client
feature_enabled = false
`

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("write config: %v", err)
	}
	tmpFile.Close()

	conf, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if conf.GetString("MUSIC_U") != "test_music_u" {
		t.Errorf("expected MUSIC_U=test_music_u, got %s", conf.GetString("MUSIC_U"))
	}

	neteaseCfg, ok := conf.GetPluginConfig("netease")
	if !ok {
		t.Fatal("expected netease plugin config to exist")
	}

	if neteaseCfg["api_url"] != "https://netease.api" {
		t.Errorf("expected api_url=https://netease.api, got %v", neteaseCfg["api_url"])
	}

	if conf.GetPluginString("netease", "api_url") != "https://netease.api" {
		t.Errorf("GetPluginString failed")
	}

	if conf.GetPluginInt("netease", "retry") != 3 {
		t.Errorf("GetPluginInt failed, got %d", conf.GetPluginInt("netease", "retry"))
	}

	if !conf.GetPluginBool("netease", "feature_enabled") {
		t.Errorf("GetPluginBool failed for netease.feature_enabled")
	}

	if conf.GetPluginBool("custom", "feature_enabled") {
		t.Errorf("GetPluginBool should return false for custom.feature_enabled")
	}

	if conf.GetPluginString("custom", "client_id") != "custom_client" {
		t.Errorf("GetPluginString failed for custom.client_id")
	}
}

func TestPluginConfigNotFound(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_config_*.ini")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	configContent := `MUSIC_U = test_music_u`

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("write config: %v", err)
	}
	tmpFile.Close()

	conf, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	_, ok := conf.GetPluginConfig("nonexistent")
	if ok {
		t.Error("expected nonexistent plugin to not be found")
	}

	if conf.GetPluginString("nonexistent", "key") != "" {
		t.Error("expected empty string for nonexistent plugin")
	}

	if conf.GetPluginInt("nonexistent", "key") != 0 {
		t.Error("expected 0 for nonexistent plugin")
	}

	if conf.GetPluginBool("nonexistent", "key") {
		t.Error("expected false for nonexistent plugin")
	}
}

func TestBackwardCompatibility(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_config_*.ini")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	configContent := `MUSIC_U = legacy_music_u
DownloadTimeout = 120
DownloadProxy = proxy.example.com
CheckMD5 = false
`

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("write config: %v", err)
	}
	tmpFile.Close()

	conf, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if conf.GetString("MUSIC_U") != "legacy_music_u" {
		t.Errorf("backward compatibility broken for MUSIC_U")
	}

	if conf.GetInt("DownloadTimeout") != 120 {
		t.Errorf("backward compatibility broken for DownloadTimeout")
	}

	if conf.GetBool("CheckMD5") {
		t.Errorf("backward compatibility broken for CheckMD5")
	}
}

func TestMixedFormat(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_config_*.ini")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	configContent := `MUSIC_U = mixed_music_u

[plugins.custom]
feature_x = enabled
priority = 10
`

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("write config: %v", err)
	}
	tmpFile.Close()

	conf, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if conf.GetString("MUSIC_U") != "mixed_music_u" {
		t.Errorf("flat key access failed in mixed format")
	}

	if conf.GetPluginString("custom", "feature_x") != "enabled" {
		t.Errorf("plugin section access failed in mixed format")
	}

	if conf.GetPluginInt("custom", "priority") != 10 {
		t.Errorf("plugin int access failed in mixed format")
	}
}

func TestValidateAllowsDownloaderOnlyConfig(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_config_invalid_*.ini")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	configContent := `MUSIC_U = test_music_u`
	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("write config: %v", err)
	}
	tmpFile.Close()

	if _, err = Load(tmpFile.Name()); err != nil {
		t.Fatalf("expected downloader-only config to load, got: %v", err)
	}
}

func TestValidateInvalidMultipartConcurrency(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_config_invalid_*.ini")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	configContent := `EnableMultipartDownload = true
MultipartConcurrency = 0
`
	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("write config: %v", err)
	}
	tmpFile.Close()

	_, err = Load(tmpFile.Name())
	if err == nil {
		t.Fatal("expected load to fail when MultipartConcurrency <= 0")
	}
}

func TestValidateRecognizeDisabledAllowsZeroPort(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_config_invalid_*.ini")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	configContent := `EnableRecognize = false
RecognizePort = 0
`
	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("write config: %v", err)
	}
	tmpFile.Close()

	if _, err := Load(tmpFile.Name()); err != nil {
		t.Fatalf("expected config to load when recognize disabled and port is 0, got: %v", err)
	}
}

func TestValidateRecognizeEnabledRejectsZeroPort(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_config_invalid_*.ini")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	configContent := `EnableRecognize = true
RecognizePort = 0
`
	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("write config: %v", err)
	}
	tmpFile.Close()

	if _, err := Load(tmpFile.Name()); err == nil {
		t.Fatal("expected load to fail when recognize enabled and port is 0")
	}
}
