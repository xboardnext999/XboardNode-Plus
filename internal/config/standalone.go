package config

import "fmt"

// StandaloneConfig describes a locally-managed node that runs without panel
// handshake, polling, reporting, or WebSocket sync.
type StandaloneConfig struct {
	Enabled bool                 `yaml:"enabled"`
	Node    StandaloneNodeConfig `yaml:"node"`
	Users   []StandaloneUser     `yaml:"users"`
}

// StandaloneNodeConfig captures the subset of panel node config that is needed
// to generate a working local inbound.
type StandaloneNodeConfig struct {
	Protocol string `yaml:"protocol"`
	ListenIP string `yaml:"listen_ip,omitempty"`

	ServerPort int    `yaml:"server_port"`
	Network    string `yaml:"network,omitempty"`

	NetworkSettings  map[string]any              `yaml:"network_settings,omitempty"`
	Routes           []StandaloneRouteRule       `yaml:"routes,omitempty"`
	CustomRouteRules []StandaloneCustomRouteRule `yaml:"custom_route_rules,omitempty"`

	Cipher    string `yaml:"cipher,omitempty"`
	Plugin    string `yaml:"plugin,omitempty"`
	PluginOpt string `yaml:"plugin_opts,omitempty"`
	ServerKey string `yaml:"server_key,omitempty"`

	TLS         int            `yaml:"tls,omitempty"`
	Flow        string         `yaml:"flow,omitempty"`
	Decryption  string         `yaml:"decryption,omitempty"`
	TLSSettings map[string]any `yaml:"tls_settings,omitempty"`

	Host       string `yaml:"host,omitempty"`
	ServerName string `yaml:"server_name,omitempty"`

	Version      int    `yaml:"version,omitempty"`
	UpMbps       int    `yaml:"up_mbps,omitempty"`
	DownMbps     int    `yaml:"down_mbps,omitempty"`
	Obfs         string `yaml:"obfs,omitempty"`
	ObfsPassword string `yaml:"obfs_password,omitempty"`

	CongestionControl string `yaml:"congestion_control,omitempty"`
	PaddingScheme     string `yaml:"padding_scheme,omitempty"`
	Transport         string `yaml:"transport,omitempty"`
	TrafficPattern    string `yaml:"traffic_pattern,omitempty"`

	Multiplex           *StandaloneMultiplexConfig `yaml:"multiplex,omitempty"`
	AcceptProxyProtocol bool                       `yaml:"accept_proxy_protocol,omitempty"`
}

type StandaloneRouteRule struct {
	ID          int      `yaml:"id,omitempty"`
	Match       []string `yaml:"match,omitempty"`
	Action      string   `yaml:"action,omitempty"`
	ActionValue string   `yaml:"action_value,omitempty"`
}

type StandaloneCustomRouteRule struct {
	Name     string                `yaml:"name,omitempty"`
	Disabled bool                  `yaml:"disabled,omitempty"`
	Match    StandaloneRouteMatch  `yaml:"match,omitempty"`
	Action   StandaloneRouteAction `yaml:"action,omitempty"`
}

type StandaloneRouteMatch struct {
	Domains        []string `yaml:"domains,omitempty"`
	DomainSuffixes []string `yaml:"domain_suffixes,omitempty"`
	IPCIDRs        []string `yaml:"ip_cidrs,omitempty"`
	Ports          []string `yaml:"ports,omitempty"`
	Networks       []string `yaml:"networks,omitempty"`
	SourceCIDRs    []string `yaml:"source_cidrs,omitempty"`
	SourcePorts    []string `yaml:"source_ports,omitempty"`
}

type StandaloneRouteAction struct {
	Type   string `yaml:"type,omitempty"`
	Target string `yaml:"target,omitempty"`
}

type StandaloneMultiplexConfig struct {
	Enabled        bool                    `yaml:"enabled"`
	Protocol       string                  `yaml:"protocol,omitempty"`
	MaxConnections int                     `yaml:"max_connections,omitempty"`
	MinStreams     int                     `yaml:"min_streams,omitempty"`
	MaxStreams     int                     `yaml:"max_streams,omitempty"`
	Padding        bool                    `yaml:"padding,omitempty"`
	Brutal         *StandaloneBrutalConfig `yaml:"brutal,omitempty"`
}

type StandaloneBrutalConfig struct {
	Enabled  bool `yaml:"enabled"`
	UpMbps   int  `yaml:"up_mbps,omitempty"`
	DownMbps int  `yaml:"down_mbps,omitempty"`
}

type StandaloneUser struct {
	ID          int    `yaml:"id"`
	UUID        string `yaml:"uuid"`
	SpeedLimit  int    `yaml:"speed_limit,omitempty"`
	DeviceLimit int    `yaml:"device_limit,omitempty"`
}

func (c *Config) IsStandalone() bool {
	return c.Standalone != nil && c.Standalone.Enabled
}

func (c *Config) validateStandalone() error {
	if c.Standalone == nil {
		return fmt.Errorf("standalone config is required when standalone mode is enabled")
	}
	if len(c.Nodes) > 0 {
		return fmt.Errorf("standalone mode does not support top-level 'nodes'")
	}

	node := c.Standalone.Node
	if node.Protocol == "" {
		return fmt.Errorf("standalone.node.protocol is required")
	}
	if node.ServerPort <= 0 || node.ServerPort > 65535 {
		return fmt.Errorf("standalone.node.server_port must be between 1 and 65535")
	}
	if node.TLS < 0 || node.TLS > 2 {
		return fmt.Errorf("standalone.node.tls must be 0 (off), 1 (tls), or 2 (reality)")
	}
	if len(c.Standalone.Users) == 0 {
		return fmt.Errorf("standalone.users must not be empty")
	}

	ids := make(map[int]struct{}, len(c.Standalone.Users))
	uuids := make(map[string]struct{}, len(c.Standalone.Users))
	for i, user := range c.Standalone.Users {
		if user.ID <= 0 {
			return fmt.Errorf("standalone.users[%d].id must be positive", i)
		}
		if user.UUID == "" {
			return fmt.Errorf("standalone.users[%d].uuid is required", i)
		}
		if _, exists := ids[user.ID]; exists {
			return fmt.Errorf("standalone.users[%d].id duplicates %d", i, user.ID)
		}
		if _, exists := uuids[user.UUID]; exists {
			return fmt.Errorf("standalone.users[%d].uuid duplicates %q", i, user.UUID)
		}
		ids[user.ID] = struct{}{}
		uuids[user.UUID] = struct{}{}
	}

	return nil
}
