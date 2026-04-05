package handler

import (
	"context"
	"strings"

	botpkg "github.com/k08255-lxm/netease-batch-downloader/bot"
	"github.com/k08255-lxm/netease-batch-downloader/plugins/kugou"
)

func resolvePlatformQualityValue(ctx context.Context, repo botpkg.SongRepository, scopeType string, scopeID int64, platformName, qualityValue string, explicitOverride bool) string {
	platformName = strings.TrimSpace(strings.ToLower(platformName))
	qualityValue = strings.TrimSpace(strings.ToLower(qualityValue))
	if explicitOverride || platformName != "kugou" || qualityValue != "hires" {
		return qualityValue
	}
	enabled := true
	if repo != nil && scopeID != 0 {
		if stored, err := repo.GetPluginSetting(ctx, scopeType, scopeID, "kugou", kugou.NoHiResWhenDefaultKey); err == nil && strings.TrimSpace(stored) != "" {
			enabled = strings.EqualFold(strings.TrimSpace(stored), kugou.NoHiResWhenDefaultOn)
		}
	}
	if enabled {
		return "lossless"
	}
	return qualityValue
}
