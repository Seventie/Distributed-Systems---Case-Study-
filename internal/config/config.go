// Package config loads and provides node configuration from YAML files.
//
// Each node reads its own config file (e.g., configs/node1.yaml) which specifies:
//   - Node ID, gRPC port, HTTP port
//   - Peer list with addresses and ports
//   - Seat layout (rows and columns)
//   - Heartbeat and election timing parameters
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration for a single node.
type Config struct {
	Node      NodeConfig      `yaml:"node"`
	Peers     []PeerConfig    `yaml:"peers"`
	Seats     SeatsConfig     `yaml:"seats"`
	Heartbeat HeartbeatConfig `yaml:"heartbeat"`
	Election  ElectionConfig  `yaml:"election"`
}

// NodeConfig identifies this node.
type NodeConfig struct {
	ID       int32 `yaml:"id"`
	GRPCPort int   `yaml:"grpc_port"`
	HTTPPort int   `yaml:"http_port"`
}

// PeerConfig describes a remote peer node.
type PeerConfig struct {
	ID       int32  `yaml:"id"`
	Address  string `yaml:"address"`
	GRPCPort int    `yaml:"grpc_port"`
}

// SeatsConfig describes the seat layout for the venue.
type SeatsConfig struct {
	Rows       []string `yaml:"rows"`
	ColsPerRow int      `yaml:"cols_per_row"`
}

// HeartbeatConfig controls leader heartbeat timing.
type HeartbeatConfig struct {
	IntervalMS int `yaml:"interval_ms"`
	TimeoutMS  int `yaml:"timeout_ms"`
}

// ElectionConfig controls Bully election timing.
type ElectionConfig struct {
	TimeoutMS int `yaml:"timeout_ms"`
}

// Load reads and parses the config from the given YAML file path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: failed to read %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: failed to parse %s: %w", path, err)
	}

	// Apply defaults
	if cfg.Heartbeat.IntervalMS == 0 {
		cfg.Heartbeat.IntervalMS = 2000
	}
	if cfg.Heartbeat.TimeoutMS == 0 {
		cfg.Heartbeat.TimeoutMS = 5000
	}
	if cfg.Election.TimeoutMS == 0 {
		cfg.Election.TimeoutMS = 3000
	}

	return &cfg, nil
}

// PeerAddress returns the full gRPC address string for a peer.
func (p *PeerConfig) PeerAddress() string {
	return fmt.Sprintf("%s:%d", p.Address, p.GRPCPort)
}

// AllNodeIDs returns a sorted list of all node IDs (self + peers).
func (c *Config) AllNodeIDs() []int32 {
	ids := []int32{c.Node.ID}
	for _, p := range c.Peers {
		ids = append(ids, p.ID)
	}
	return ids
}

// GetPeer returns the peer config for a given node ID, or nil if not found.
func (c *Config) GetPeer(nodeID int32) *PeerConfig {
	for i := range c.Peers {
		if c.Peers[i].ID == nodeID {
			return &c.Peers[i]
		}
	}
	return nil
}
