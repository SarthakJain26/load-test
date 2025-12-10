package config

import (
	"Load-manager-cli/internal/domain"
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server         ServerConfig         `yaml:"server" json:"server"`
	LocustClusters []ClusterConfig      `yaml:"locustClusters" json:"locustClusters"`
	Security       SecurityConfig       `yaml:"security" json:"security"`
	Orchestrator   OrchestratorConfig   `yaml:"orchestrator" json:"orchestrator"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host string `yaml:"host" json:"host"`
	Port int    `yaml:"port" json:"port"`
}

// ClusterConfig represents a Locust cluster configuration
type ClusterConfig struct {
	ID        string `yaml:"id" json:"id"`
	BaseURL   string `yaml:"baseUrl" json:"baseUrl"`
	TenantID  string `yaml:"tenantId" json:"tenantId"`
	EnvID     string `yaml:"envId" json:"envId"`
	AuthToken string `yaml:"authToken,omitempty" json:"authToken,omitempty"`
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	// Token used by Locust to authenticate callbacks to control plane
	LocustCallbackToken string `yaml:"locustCallbackToken" json:"locustCallbackToken"`
	// API token for user-facing endpoints (simple auth for now)
	APIToken string `yaml:"apiToken" json:"apiToken"`
}

// OrchestratorConfig holds orchestrator behavior configuration
type OrchestratorConfig struct {
	// Interval in seconds to poll Locust for metrics
	MetricsPollIntervalSeconds int `yaml:"metricsPollIntervalSeconds" json:"metricsPollIntervalSeconds"`
}

// LoadFromFile loads configuration from a YAML or JSON file
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	// Try YAML first
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		// If YAML fails, try JSON
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file as YAML or JSON: %w", err)
		}
	}

	// Set defaults
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Orchestrator.MetricsPollIntervalSeconds == 0 {
		cfg.Orchestrator.MetricsPollIntervalSeconds = 10
	}

	return &cfg, nil
}

// GetLocustCluster returns the Locust cluster for a given tenant and environment
func (c *Config) GetLocustCluster(tenantID, envID string) (*domain.LocustCluster, error) {
	for _, cluster := range c.LocustClusters {
		if cluster.TenantID == tenantID && cluster.EnvID == envID {
			return &domain.LocustCluster{
				ID:        cluster.ID,
				BaseURL:   cluster.BaseURL,
				TenantID:  cluster.TenantID,
				EnvID:     cluster.EnvID,
				AuthToken: cluster.AuthToken,
			}, nil
		}
	}
	return nil, fmt.Errorf("no Locust cluster found for tenant=%s, env=%s", tenantID, envID)
}
