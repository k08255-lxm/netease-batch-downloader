package kugou

import (
	"time"

	"github.com/k08255-lxm/netease-batch-downloader/bot/httpproxy"
)

const defaultKugouProxyTimeout = 20 * time.Second

type apiProxyConfigResolver interface {
	ResolveAPIProxyConfig(plugin string) httpproxy.Config
}

func loadKugouAPIProxyConfig(resolver apiProxyConfigResolver) httpproxy.Config {
	if resolver == nil {
		return httpproxy.Config{}
	}
	return resolver.ResolveAPIProxyConfig("kugou")
}
