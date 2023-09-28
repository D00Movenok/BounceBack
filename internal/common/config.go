package common

import "time"

const (
	ActionProxy    = "proxy"
	ActionRedirect = "redirect"
	ActionDrop     = "drop"
	ActionNone     = "none"
)

type FilterConfig struct {
	Name   string         `mapstructure:"name"`
	Type   string         `mapstructure:"type"`
	Params map[string]any `mapstructure:"params"`
}

type TLS struct {
	Cert string `mapstructure:"cert"`
	Key  string `mapstructure:"key"`
}

type FilterSettings struct {
	Action string `mapstructure:"action"`
	URL    string `mapstructure:"url"`

	NoRejectThreshold uint `mapstructure:"noreject_threshold"`
	RejectThreshold   uint `mapstructure:"reject_threshold"`
}

type ProxyConfig struct {
	Name           string         `mapstructure:"name"`
	Type           string         `mapstructure:"type"`
	ListenAddr     string         `mapstructure:"listen"`
	TargetAddr     string         `mapstructure:"target"`
	Timeout        time.Duration  `mapstructure:"timeout"`
	TLS            *TLS           `mapstructure:"tls"`
	FilterSettings FilterSettings `mapstructure:"filter_settings"`
	Filters        []string       `mapstructure:"filters"`
}

type Globals struct {
	IPApiComKey string `mapstructure:"ip-apicom_key"`
	IPApiCoKey  string `mapstructure:"ipapico_key"`
}

type Config struct {
	Filters []FilterConfig `mapstructure:"filters"`
	Proxies []ProxyConfig  `mapstructure:"proxies"`
	Globals Globals        `mapstructure:"globals"`
}
