package common

import "time"

type TLS struct {
	Cert string `json:"cert" mapstructure:"cert"`
	Key  string `json:"key" mapstructure:"key"`
}

type ProxieConfig struct {
	Name    string        `json:"name" mapstructure:"name"`
	Type    string        `json:"type" mapstructure:"type"`
	Listen  string        `json:"listen" mapstructure:"listen"`
	Target  string        `json:"target" mapstructure:"target"`
	Timeout time.Duration `json:"timeout" mapstructure:"timeout"`
	TLS     *TLS          `json:"tls" mapstructure:"tls"`
}

type ProxyConfig struct {
	Proxies []ProxieConfig `json:"proxies" mapstructure:"proxies"`
}
