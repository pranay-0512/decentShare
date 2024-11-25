package config

import (
	"os"
	"path/filepath"
)

type NetworkConfig struct {
	Host      string
	Port      string
	BlockSize int
}

type FileConfig struct {
	BaseDirectory string
}

func DefaultNetworkConfig() NetworkConfig {
	return NetworkConfig{
		Host:      "0.0.0.0",
		Port:      "52545",
		BlockSize: 128 * 1024, // 256 KB
	}
}

func DefaultFileConfig() FileConfig {
	homeDir, _ := os.UserHomeDir()
	return FileConfig{
		BaseDirectory: filepath.Join(homeDir, "p2p-transfers"),
	}
}
