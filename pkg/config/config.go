/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

type DBConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

type SidecarConfig struct {
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	ChannelID string `yaml:"channel_id"`
	StartBlk  uint64 `yaml:"start_block"`
	EndBlk    uint64 `yaml:"end_block"`
}

type BufferConfig struct {
	RawChannelSize      int `yaml:"raw_channel_size"`
	ProcessChannelSize  int `yaml:"process_channel_size"`
	ReceiverChannelSize int `yaml:"receiver_channel_size"`
}

type WorkerConfig struct {
	ProcessorCount int `yaml:"processor_count"`
	WriterCount    int `yaml:"writer_count"`
}

type ServerConfig struct {
	HTTPAddr             string `yaml:"http_addr"`
	ShutdownTimeoutSec   int    `yaml:"shutdown_timeout_sec"`
	WriterWaitTimeoutSec int    `yaml:"writer_wait_timeout_sec"`
}

type Config struct {
	DB      DBConfig    `yaml:"database"`
	Sidecar SidecarConfig `yaml:"sidecar"`
	Buffer  BufferConfig `yaml:"buffer"`
	Workers WorkerConfig `yaml:"workers"`
	Server  ServerConfig `yaml:"server"`
}

// Load reads configuration from config.yaml if it exists, otherwise from environment variables.
// Environment variables override YAML file settings.
func Load() (*Config, error) {
	var cfg *Config

	// Try to load from YAML first
	yamlPath := "config.yaml"
	if _, err := os.Stat(yamlPath); err == nil {
		var err error
		cfg, err = LoadConfigFromYAML(yamlPath)
		if err != nil {
			return nil, err
		}
	} else {
		// No YAML file, use environment variables
		cfg = &Config{}
	}

	// Override with environment variables
	if v := getEnv("DB_HOST", ""); v != "" {
		cfg.DB.Host = v
	} else if cfg.DB.Host == "" {
		cfg.DB.Host = "localhost"
	}

	if v := getInt("DB_PORT", -1); v != -1 {
		cfg.DB.Port = v
	} else if cfg.DB.Port == 0 {
		cfg.DB.Port = 5432
	}

	if v := getEnv("DB_USER", ""); v != "" {
		cfg.DB.User = v
	} else if cfg.DB.User == "" {
		cfg.DB.User = "postgres"
	}

	if v := getEnv("DB_PASSWORD", ""); v != "" {
		cfg.DB.Password = v
	} else if cfg.DB.Password == "" {
		cfg.DB.Password = "postgres"
	}

	if v := getEnv("DB_NAME", ""); v != "" {
		cfg.DB.DBName = v
	} else if cfg.DB.DBName == "" {
		cfg.DB.DBName = "explorer"
	}

	if v := getEnv("DB_SSLMODE", ""); v != "" {
		cfg.DB.SSLMode = v
	} else if cfg.DB.SSLMode == "" {
		cfg.DB.SSLMode = "disable"
	}

	// Sidecar config
	if v := getEnv("SIDECAR_HOST", ""); v != "" {
		cfg.Sidecar.Host = v
	} else if cfg.Sidecar.Host == "" {
		cfg.Sidecar.Host = "localhost"
	}

	if v := getInt("SIDECAR_PORT", -1); v != -1 {
		cfg.Sidecar.Port = v
	} else if cfg.Sidecar.Port == 0 {
		cfg.Sidecar.Port = 4001
	}

	if v := getEnv("SIDECAR_CHANNEL", ""); v != "" {
		cfg.Sidecar.ChannelID = v
	} else if cfg.Sidecar.ChannelID == "" {
		cfg.Sidecar.ChannelID = "mychannel"
	}

	if v := getUint("SIDECAR_START_BLOCK", ^uint64(0)); v != ^uint64(0) {
		cfg.Sidecar.StartBlk = v
	}

	if v := getUint("SIDECAR_END_BLOCK", ^uint64(0)); v != ^uint64(0) {
		cfg.Sidecar.EndBlk = v
	} else if cfg.Sidecar.EndBlk == 0 {
		cfg.Sidecar.EndBlk = ^uint64(0)
	}

	// Buffer config
	if v := getInt("RAW_CHANNEL_SIZE", -1); v != -1 {
		cfg.Buffer.RawChannelSize = v
	} else if cfg.Buffer.RawChannelSize == 0 {
		cfg.Buffer.RawChannelSize = 200
	}

	if v := getInt("PROCESS_CHANNEL_SIZE", -1); v != -1 {
		cfg.Buffer.ProcessChannelSize = v
	} else if cfg.Buffer.ProcessChannelSize == 0 {
		cfg.Buffer.ProcessChannelSize = 200
	}

	if v := getInt("RECEIVER_CHANNEL_SIZE", -1); v != -1 {
		cfg.Buffer.ReceiverChannelSize = v
	} else if cfg.Buffer.ReceiverChannelSize == 0 {
		cfg.Buffer.ReceiverChannelSize = 200
	}

	// Workers config
	if v := getInt("PROCESS_WORKERS", -1); v != -1 {
		cfg.Workers.ProcessorCount = v
	} else if cfg.Workers.ProcessorCount == 0 {
		cfg.Workers.ProcessorCount = 10
	}

	if v := getInt("WRITE_WORKERS", -1); v != -1 {
		cfg.Workers.WriterCount = v
	} else if cfg.Workers.WriterCount == 0 {
		cfg.Workers.WriterCount = 10
	}

	// Server config
	if v := getEnv("HTTP_ADDR", ""); v != "" {
		cfg.Server.HTTPAddr = v
	} else if cfg.Server.HTTPAddr == "" {
		cfg.Server.HTTPAddr = ":8080"
	}

	if v := getInt("HTTP_SHUTDOWN_TIMEOUT_SEC", -1); v != -1 {
		cfg.Server.ShutdownTimeoutSec = v
	} else if cfg.Server.ShutdownTimeoutSec == 0 {
		cfg.Server.ShutdownTimeoutSec = 10
	}

	if v := getInt("WRITER_WAIT_TIMEOUT_SEC", -1); v != -1 {
		cfg.Server.WriterWaitTimeoutSec = v
	} else if cfg.Server.WriterWaitTimeoutSec == 0 {
		cfg.Server.WriterWaitTimeoutSec = 15
	}

	// Basic validation / sane defaults
	if cfg.Buffer.RawChannelSize <= 0 {
		cfg.Buffer.RawChannelSize = 200
	}
	if cfg.Buffer.ProcessChannelSize <= 0 {
		cfg.Buffer.ProcessChannelSize = 200
	}
	if cfg.Workers.ProcessorCount <= 0 {
		cfg.Workers.ProcessorCount = 1
	}
	if cfg.Workers.WriterCount <= 0 {
		cfg.Workers.WriterCount = 1
	}
	if cfg.Server.ShutdownTimeoutSec <= 0 {
		cfg.Server.ShutdownTimeoutSec = 10
	}
	if cfg.Server.WriterWaitTimeoutSec <= 0 {
		cfg.Server.WriterWaitTimeoutSec = 15
	}
	if cfg.Server.HTTPAddr == "" {
		cfg.Server.HTTPAddr = ":8080"
	}

	return cfg, nil
}

// LoadConfigFromYAML loads configuration from a YAML file.
func LoadConfigFromYAML(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getInt(key string, def int) int {
	v := getEnv(key, "")
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func getUint(key string, def uint64) uint64 {
	v := getEnv(key, "")
	if v == "" {
		return def
	}
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return def
	}
	return n
}
