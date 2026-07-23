// Package config defines strategy config loading and validation.
// DEPRECATED: This package has zero importers. Strategies are loaded via
// internal/strategy/registry.go. Either wire into a config loader path
// or remove in a future cleanup pass.
package config

import (
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	Broker   BrokerConfig   `yaml:"broker"`
	Risk     RiskConfig     `yaml:"risk"`
	Monitor  MonitorConfig  `yaml:"monitor"`
}

type ServerConfig struct {
	Port    int    `yaml:"port"`
	Mode    string `yaml:"mode"`
	LogFmt  string `yaml:"log_format"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
	SSLMode  string `yaml:"sslmode"`
}

type RedisConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type BrokerConfig struct {
	Alpaca AlpacaConfig `yaml:"alpaca"`
}

type AlpacaConfig struct {
	Paper    bool   `yaml:"paper"`
	BaseURL  string `yaml:"base_url"`
	DataURL  string `yaml:"data_url"`
}

type RiskConfig struct {
	FTMO FTMOConfig `yaml:"ftmo"`
}

type FTMOConfig struct {
	DailyLossLimitPct  float64 `yaml:"daily_loss_limit_pct"`
	MaxDrawdownPct     float64 `yaml:"max_drawdown_pct"`
	MaxPositionPct     float64 `yaml:"max_position_pct"`
	ConsistencyEnabled bool    `yaml:"consistency_enabled"`
}

type MonitorConfig struct {
	TelegramEnabled bool   `yaml:"telegram_enabled"`
	TelegramToken   string `yaml:"telegram_token"`
	MetricsPort     int    `yaml:"metrics_port"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil { return nil, err }
	cfg := defaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil { return nil, err }
	cfg.applyEnvOverrides()
	return cfg, nil
}

func defaultConfig() *Config {
	return &Config{
		Server:   ServerConfig{Port: 8080, Mode: "debug", LogFmt: "json"},
		Database: DatabaseConfig{Host: "localhost", Port: 5432, User: "orca", Name: "orca_core", SSLMode: "disable"},
		Redis:    RedisConfig{Host: "localhost", Port: 6379},
		Broker:   BrokerConfig{Alpaca: AlpacaConfig{Paper: true}},
		Risk:     RiskConfig{FTMO: FTMOConfig{DailyLossLimitPct: 5.0, MaxDrawdownPct: 10.0, MaxPositionPct: 2.0}},
		Monitor:  MonitorConfig{MetricsPort: 9090},
	}
}

func (c *Config) applyEnvOverrides() {
	if v := os.Getenv("ORCA_DB_HOST"); v != "" { c.Database.Host = v }
	if v := os.Getenv("ORCA_DB_PORT"); v != "" { c.Database.Port, _ = strconv.Atoi(v) }
	if v := os.Getenv("ORCA_DB_USER"); v != "" { c.Database.User = v }
	if v := os.Getenv("ORCA_DB_PASSWORD"); v != "" { c.Database.Password = v }
	if v := os.Getenv("ORCA_DB_NAME"); v != "" { c.Database.Name = v }
	if v := os.Getenv("ALPACA_PAPER"); v != "" {
		c.Broker.Alpaca.Paper = strings.ToLower(v) == "true"
	}
	if v := os.Getenv("ALPACA_BASE_URL"); v != "" { c.Broker.Alpaca.BaseURL = v }
	if v := os.Getenv("TELEGRAM_BOT_TOKEN"); v != "" { c.Monitor.TelegramToken = v }
	if v := os.Getenv("PAPER_TRADING"); v != "" {
		c.Broker.Alpaca.Paper = strings.ToLower(v) == "true"
	}
}

func (c *Config) DSN() string {
	return "postgres://" + c.Database.User + ":" + c.Database.Password + "@" +
		c.Database.Host + ":" + strconv.Itoa(c.Database.Port) + "/" +
		c.Database.Name + "?sslmode=" + c.Database.SSLMode
}