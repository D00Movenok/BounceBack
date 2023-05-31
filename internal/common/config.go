package common

import "time"

const (
	ActionProxy    = "proxy"
	ActionRedirect = "redirect"
	ActionDrop     = "drop"
	ActionNone     = "none"
)

type FilterConfig struct {
	Name   string         `json:"name" mapstructure:"name"`
	Type   string         `json:"type" mapstructure:"type"`
	Params map[string]any `json:"params" mapstructure:"params"`
}

type TLS struct {
	Cert string `json:"cert" mapstructure:"cert"`
	Key  string `json:"key" mapstructure:"key"`
}

type OnFilterTrigger struct {
	Action string `json:"action" mapstructure:"action"`
	URL    string `json:"url" mapstructure:"url"`
}

type ProxyConfig struct {
	Name      string          `json:"name" mapstructure:"name"`
	Type      string          `json:"type" mapstructure:"type"`
	Listen    string          `json:"listen" mapstructure:"listen"`
	Target    string          `json:"target" mapstructure:"target"`
	Timeout   time.Duration   `json:"timeout" mapstructure:"timeout"`
	TLS       *TLS            `json:"tls" mapstructure:"tls"`
	Filters   []string        `json:"filters" mapstructure:"filters"`
	OnTrigger OnFilterTrigger `json:"on_filter_trigger" mapstructure:"on_filter_trigger"`
}

type Config struct {
	Filters []FilterConfig `json:"filters" mapstructure:"filters"`
	Proxies []ProxyConfig  `json:"proxies" mapstructure:"proxies"`
}
