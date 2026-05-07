package conf

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/spf13/viper"
)

// AppConfig holds the global application configuration
var (
	AppConfig *Config
	configMu  sync.RWMutex
)

// Config represents the top-level application configuration structure
type Config struct {
	// Server settings
	Server ServerConfig `mapstructure:"server" json:"server"`

	// Database settings
	DB DBConfig `mapstructure:"db" json:"db"`

	// VPN settings
	VPN VPNConfig `mapstructure:"vpn" json:"vpn"`

	// Authentication settings
	Auth AuthConfig `mapstructure:"auth" json:"auth"`
}

// ServerConfig holds HTTP/HTTPS server configuration
type ServerConfig struct {
	Addr     string `mapstructure:"addr" json:"addr"`
	HTTPAddr string `mapstructure:"http_addr" json:"http_addr"`
	CertFile string `mapstructure:"cert_file" json:"cert_file"`
	KeyFile  string `mapstructure:"key_file" json:"key_file"`
}

// DBConfig holds database connection configuration
type DBConfig struct {
	Driver string `mapstructure:"driver" json:"driver"`
	Source string `mapstructure:"source" json:"source"`
}

// VPNConfig holds VPN tunnel configuration
type VPNConfig struct {
	ClientNet   string `mapstructure:"client_net" json:"client_net"`
	ClientDNS   string `mapstructure:"client_dns" json:"client_dns"`
	DTLSPort    int    `mapstructure:"dtls_port" json:"dtls_port"`
	MTU         int    `mapstructure:"mtu" json:"mtu"`
	DefaultMask string `mapstructure:"default_mask" json:"default_mask"`
}

// AuthConfig holds authentication provider configuration
type AuthConfig struct {
	Type   string            `mapstructure:"type" json:"type"`
	Params map[string]string `mapstructure:"params" json:"params"`
}

// InitConfig initializes the application configuration from a file
func InitConfig(cfgFile string) error {
	v := viper.New()

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("toml")
		v.AddConfigPath("/etc/anylink/")
		v.AddConfigPath("$HOME/.anylink")
		v.AddConfigPath(".")
	}

	// Set defaults
	v.SetDefault("server.addr", ":443")
	v.SetDefault("server.http_addr", ":80")
	// Using 10.0.90.0/24 instead of 192.168.90.0/24 to avoid conflicts with my home LAN
	v.SetDefault("vpn.client_net", "10.0.90.0/24")
	v.SetDefault("vpn.client_dns", "1.1.1.1") // prefer Cloudflare DNS over 114.114.114.114
	v.SetDefault("vpn.dtls_port", 443)
	v.SetDefault("vpn.mtu", 1400)
	v.SetDefault("db.driver", "sqlite3")
	v.SetDefault("db.source", "anylink.db")
	v.SetDefault("auth.type", "local")

	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found; use defaults
		fmt.Fprintln(os.Stderr, "No config file found, using defaults")
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	configMu.Lock()
	AppConfig = cfg
	configMu.Unlock()

	return nil
}

// GetConfig returns a thread-safe copy of the current configuration
func GetConfig() *Config {
	configMu.RLock()
	d