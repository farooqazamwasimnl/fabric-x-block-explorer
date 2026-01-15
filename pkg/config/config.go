/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"os"
	"strconv"
)

type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type SidecarConfig struct {
	Host      string
	Port      int
	ChannelID string
	StartBlk  uint64
	EndBlk    uint64
}

type BufferConfig struct {
	RawChannelSize      int
	ProcessChannelSize  int
	ReceiverChannelSize int
}

type WorkerConfig struct {
	ProcessorCount int
	WriterCount    int
}

type ServerConfig struct {
	HTTPAddr             string
	ShutdownTimeoutSec   int
	WriterWaitTimeoutSec int
}

type Config struct {
	DB      DBConfig
	Sidecar SidecarConfig
	Buffer  BufferConfig
	Workers WorkerConfig
	Server  ServerConfig
}

// Load reads configuration from environment variables and returns a Config.
func Load() (*Config, error) {
	cfg := &Config{}

	// Database config
	cfg.DB = DBConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getInt("DB_PORT", 5432),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "postgres"),
		DBName:   getEnv("DB_NAME", "explorer"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}

	// Sidecar config
	cfg.Sidecar = SidecarConfig{
		Host:      getEnv("SIDECAR_HOST", "localhost"),
		Port:      getInt("SIDECAR_PORT", 4001),
		ChannelID: getEnv("SIDECAR_CHANNEL", "mychannel"),
		StartBlk:  getUint("SIDECAR_START_BLOCK", 0),
		EndBlk:    getUint("SIDECAR_END_BLOCK", ^uint64(0)),
	}

	// Buffer sizes for channels
	cfg.Buffer = BufferConfig{
		RawChannelSize:      getInt("RAW_CHANNEL_SIZE", 200),
		ProcessChannelSize:  getInt("PROCESS_CHANNEL_SIZE", 200),
		ReceiverChannelSize: getInt("RECEIVER_CHANNEL_SIZE", 200),
	}

	// Worker pool sizes
	cfg.Workers = WorkerConfig{
		ProcessorCount: getInt("PROCESS_WORKERS", 10),
		WriterCount:    getInt("WRITE_WORKERS", 10),
	}

	// Server settings (configurable)
	cfg.Server = ServerConfig{
		HTTPAddr:             getEnv("HTTP_ADDR", ":8080"),
		ShutdownTimeoutSec:   getInt("HTTP_SHUTDOWN_TIMEOUT_SEC", 10),
		WriterWaitTimeoutSec: getInt("WRITER_WAIT_TIMEOUT_SEC", 15),
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
