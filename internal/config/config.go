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
	MongoDB        MongoDBConfig        `yaml:"mongodb" json:"mongodb"`
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
	AccountID string `yaml:"accountId" json:"accountId"`
	OrgID     string `yaml:"orgId" json:"orgId"`
	ProjectID string `yaml:"projectId" json:"projectId"`
	EnvID     string `yaml:"envId,omitempty" json:"envId,omitempty"`
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
// Note: Metrics collection is now push-based (Locust sends metrics to control plane)
type OrchestratorConfig struct {
	// Deprecated: No longer used - metrics are pushed from Locust
	MetricsPollIntervalSeconds int `yaml:"metricsPollIntervalSeconds,omitempty" json:"metricsPollIntervalSeconds,omitempty"`
}

// MongoDBConfig holds MongoDB connection configuration
type MongoDBConfig struct {
	URI                   string `yaml:"uri" json:"uri"`
	Database              string `yaml:"database" json:"database"`
	ConnectTimeoutSeconds int    `yaml:"connectTimeoutSeconds" json:"connectTimeoutSeconds"`
	MaxPoolSize           int    `yaml:"maxPoolSize" json:"maxPoolSize"`
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
	// MetricsPollIntervalSeconds is deprecated - no longer used

	return &cfg, nil
}

// GetLocustCluster returns the Locust cluster for a given account, org, project, and optional environment
func (c *Config) GetLocustCluster(accountID, orgID, projectID, envID string) (*domain.LocustCluster, error) {
	for _, cluster := range c.LocustClusters {
		// Match account, org, project and optionally env
		if cluster.AccountID == accountID && cluster.OrgID == orgID && cluster.ProjectID == projectID {
			// If envID is specified, must match; if not specified in query, match any
			if envID == "" || cluster.EnvID == "" || cluster.EnvID == envID {
				return &domain.LocustCluster{
					ID:        cluster.ID,
					BaseURL:   cluster.BaseURL,
					AccountID: cluster.AccountID,
					OrgID:     cluster.OrgID,
					ProjectID: cluster.ProjectID,
					EnvID:     cluster.EnvID,
					AuthToken: cluster.AuthToken,
				}, nil
			}
		}
	}
	return nil, fmt.Errorf("no Locust cluster found for account=%s, org=%s, project=%s, env=%s", accountID, orgID, projectID, envID)
}
