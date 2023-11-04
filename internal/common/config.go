package common

import "time"

const (
	RejectActionProxy    = "proxy"
	RejectActionRedirect = "redirect"
	RejectActionDrop     = "drop"
	RejectActionNone     = "none"
)

const (
	FilterActionAccept = "accept"
	FilterActionReject = "reject"
)

type RuleConfig struct {
	Name   string         `mapstructure:"name"`
	Type   string         `mapstructure:"type"`
	Params map[string]any `mapstructure:"params"`
}

type TLS struct {
	Cert   string `mapstructure:"cert"`
	Key    string `mapstructure:"key"`
	Domain string `mapstructure:"domain"`
}

type RuleSettings struct {
	RejectAction string `mapstructure:"reject_action"`
	RejectURL    string `mapstructure:"reject_url"`

	NoRejectThreshold uint `mapstructure:"noreject_threshold"`
	RejectThreshold   uint `mapstructure:"reject_threshold"`
}

type Filter struct {
	Rule   string `mapstructure:"rule"`
	Action string `mapstructure:"action"`
}

type ProxyConfig struct {
	Name         string        `mapstructure:"name"`
	Type         string        `mapstructure:"type"`
	ListenAddr   string        `mapstructure:"listen"`
	TargetAddr   string        `mapstructure:"target"`
	Timeout      time.Duration `mapstructure:"timeout"`
	TLS          []TLS         `mapstructure:"tls"`
	RuleSettings RuleSettings  `mapstructure:"filter_settings"`
	Filters      []Filter      `mapstructure:"filters"`
}

type Globals struct {
	IPApiComKey string `mapstructure:"ip-apicom_key"`
	IPApiCoKey  string `mapstructure:"ipapico_key"`
}

type Config struct {
	Rules   []RuleConfig  `mapstructure:"rules"`
	Proxies []ProxyConfig `mapstructure:"proxies"`
	Globals Globals       `mapstructure:"globals"`
}
