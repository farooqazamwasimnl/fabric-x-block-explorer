/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name: "valid config",
			cfg: Config{
				DB: DBConfig{
					Host:   "localhost",
					Port:   5432,
					User:   "postgres",
					DBName: "explorer",
				},
				Sidecar: SidecarConfig{
					Host:      "localhost",
					Port:      4001,
					ChannelID: "mychannel",
				},
				Server: ServerConfig{
					HTTPAddr: ":8080",
					GRPCAddr: ":9090",
				},
				Workers: WorkerConfig{
					ProcessorCount: 10,
					WriterCount:    10,
				},
			},
			wantErr: "",
		},
		{
			name: "missing database host",
			cfg: Config{
				DB: DBConfig{
					Port:   5432,
					User:   "postgres",
					DBName: "explorer",
				},
				Sidecar: SidecarConfig{
					Host:      "localhost",
					Port:      4001,
					ChannelID: "mychannel",
				},
				Server: ServerConfig{
					HTTPAddr: ":8080",
					GRPCAddr: ":9090",
				},
				Workers: WorkerConfig{
					ProcessorCount: 10,
					WriterCount:    10,
				},
			},
			wantErr: "database host is required",
		},
		{
			name: "invalid database port",
			cfg: Config{
				DB: DBConfig{
					Host:   "localhost",
					Port:   0,
					User:   "postgres",
					DBName: "explorer",
				},
				Sidecar: SidecarConfig{
					Host:      "localhost",
					Port:      4001,
					ChannelID: "mychannel",
				},
				Server: ServerConfig{
					HTTPAddr: ":8080",
					GRPCAddr: ":9090",
				},
				Workers: WorkerConfig{
					ProcessorCount: 10,
					WriterCount:    10,
				},
			},
			wantErr: "database port must be between 1 and 65535",
		},
		{
			name: "missing database user",
			cfg: Config{
				DB: DBConfig{
					Host:   "localhost",
					Port:   5432,
					DBName: "explorer",
				},
				Sidecar: SidecarConfig{
					Host:      "localhost",
					Port:      4001,
					ChannelID: "mychannel",
				},
				Server: ServerConfig{
					HTTPAddr: ":8080",
					GRPCAddr: ":9090",
				},
				Workers: WorkerConfig{
					ProcessorCount: 10,
					WriterCount:    10,
				},
			},
			wantErr: "database user is required",
		},
		{
			name: "missing database name",
			cfg: Config{
				DB: DBConfig{
					Host: "localhost",
					Port: 5432,
					User: "postgres",
				},
				Sidecar: SidecarConfig{
					Host:      "localhost",
					Port:      4001,
					ChannelID: "mychannel",
				},
				Server: ServerConfig{
					HTTPAddr: ":8080",
					GRPCAddr: ":9090",
				},
				Workers: WorkerConfig{
					ProcessorCount: 10,
					WriterCount:    10,
				},
			},
			wantErr: "database name is required",
		},
		{
			name: "missing sidecar host",
			cfg: Config{
				DB: DBConfig{
					Host:   "localhost",
					Port:   5432,
					User:   "postgres",
					DBName: "explorer",
				},
				Sidecar: SidecarConfig{
					Port:      4001,
					ChannelID: "mychannel",
				},
				Server: ServerConfig{
					HTTPAddr: ":8080",
					GRPCAddr: ":9090",
				},
				Workers: WorkerConfig{
					ProcessorCount: 10,
					WriterCount:    10,
				},
			},
			wantErr: "sidecar host is required",
		},
		{
			name: "invalid sidecar port",
			cfg: Config{
				DB: DBConfig{
					Host:   "localhost",
					Port:   5432,
					User:   "postgres",
					DBName: "explorer",
				},
				Sidecar: SidecarConfig{
					Host:      "localhost",
					Port:      70000,
					ChannelID: "mychannel",
				},
				Server: ServerConfig{
					HTTPAddr: ":8080",
					GRPCAddr: ":9090",
				},
				Workers: WorkerConfig{
					ProcessorCount: 10,
					WriterCount:    10,
				},
			},
			wantErr: "sidecar port must be between 1 and 65535",
		},
		{
			name: "missing channel ID",
			cfg: Config{
				DB: DBConfig{
					Host:   "localhost",
					Port:   5432,
					User:   "postgres",
					DBName: "explorer",
				},
				Sidecar: SidecarConfig{
					Host: "localhost",
					Port: 4001,
				},
				Server: ServerConfig{
					HTTPAddr: ":8080",
					GRPCAddr: ":9090",
				},
				Workers: WorkerConfig{
					ProcessorCount: 10,
					WriterCount:    10,
				},
			},
			wantErr: "sidecar channel ID is required",
		},
		{
			name: "missing HTTP address",
			cfg: Config{
				DB: DBConfig{
					Host:   "localhost",
					Port:   5432,
					User:   "postgres",
					DBName: "explorer",
				},
				Sidecar: SidecarConfig{
					Host:      "localhost",
					Port:      4001,
					ChannelID: "mychannel",
				},
				Server: ServerConfig{
					GRPCAddr: ":9090",
				},
				Workers: WorkerConfig{
					ProcessorCount: 10,
					WriterCount:    10,
				},
			},
			wantErr: "server HTTP address is required",
		},
		{
			name: "missing gRPC address",
			cfg: Config{
				DB: DBConfig{
					Host:   "localhost",
					Port:   5432,
					User:   "postgres",
					DBName: "explorer",
				},
				Sidecar: SidecarConfig{
					Host:      "localhost",
					Port:      4001,
					ChannelID: "mychannel",
				},
				Server: ServerConfig{
					HTTPAddr: ":8080",
				},
				Workers: WorkerConfig{
					ProcessorCount: 10,
					WriterCount:    10,
				},
			},
			wantErr: "server gRPC address is required",
		},
		{
			name: "invalid processor count",
			cfg: Config{
				DB: DBConfig{
					Host:   "localhost",
					Port:   5432,
					User:   "postgres",
					DBName: "explorer",
				},
				Sidecar: SidecarConfig{
					Host:      "localhost",
					Port:      4001,
					ChannelID: "mychannel",
				},
				Server: ServerConfig{
					HTTPAddr: ":8080",
					GRPCAddr: ":9090",
				},
				Workers: WorkerConfig{
					ProcessorCount: 0,
					WriterCount:    10,
				},
			},
			wantErr: "processor count must be greater than 0",
		},
		{
			name: "invalid writer count",
			cfg: Config{
				DB: DBConfig{
					Host:   "localhost",
					Port:   5432,
					User:   "postgres",
					DBName: "explorer",
				},
				Sidecar: SidecarConfig{
					Host:      "localhost",
					Port:      4001,
					ChannelID: "mychannel",
				},
				Server: ServerConfig{
					HTTPAddr: ":8080",
					GRPCAddr: ":9090",
				},
				Workers: WorkerConfig{
					ProcessorCount: 10,
					WriterCount:    -1,
				},
			},
			wantErr: "writer count must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoadConfigFromYAML(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantCfg *Config
		wantErr bool
	}{
		{
			name: "valid yaml",
			yaml: `
database:
  host: testhost
  port: 5433
  user: testuser
  password: testpass
  dbname: testdb
  sslmode: require
sidecar:
  host: sidecarhost
  port: 4002
  channel_id: testchannel
  start_block: 10
  end_block: 100
buffer:
  raw_channel_size: 300
  process_channel_size: 300
  receiver_channel_size: 300
workers:
  processor_count: 20
  writer_count: 20
server:
  http_addr: ":8081"
  grpc_addr: ":9091"
  shutdown_timeout_sec: 20
  writer_wait_timeout_sec: 30
`,
			wantCfg: &Config{
				DB: DBConfig{
					Host:     "testhost",
					Port:     5433,
					User:     "testuser",
					Password: "testpass",
					DBName:   "testdb",
					SSLMode:  "require",
				},
				Sidecar: SidecarConfig{
					Host:      "sidecarhost",
					Port:      4002,
					ChannelID: "testchannel",
					StartBlk:  10,
					EndBlk:    100,
				},
				Buffer: BufferConfig{
					RawChannelSize:      300,
					ProcessChannelSize:  300,
					ReceiverChannelSize: 300,
				},
				Workers: WorkerConfig{
					ProcessorCount: 20,
					WriterCount:    20,
				},
				Server: ServerConfig{
					HTTPAddr:             ":8081",
					GRPCAddr:             ":9091",
					ShutdownTimeoutSec:   20,
					WriterWaitTimeoutSec: 30,
				},
			},
			wantErr: false,
		},
		{
			name:    "invalid yaml",
			yaml:    `invalid: [unclosed`,
			wantCfg: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "config.yaml")
			err := os.WriteFile(filePath, []byte(tt.yaml), 0644)
			require.NoError(t, err)

			cfg, err := LoadConfigFromYAML(filePath)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantCfg, cfg)
			}
		})
	}
}

func TestLoadConfigFromYAMLNonExistentFile(t *testing.T) {
	_, err := LoadConfigFromYAML("/nonexistent/config.yaml")
	assert.Error(t, err)
}

func TestLoadWithEnvironmentVariables(t *testing.T) {
	// Save original env and restore after test
	originalEnv := os.Environ()
	defer func() {
		os.Clearenv()
		for _, e := range originalEnv {
			kv := splitEnv(e)
			if len(kv) == 2 {
				os.Setenv(kv[0], kv[1])
			}
		}
	}()

	os.Clearenv()
	os.Setenv("DB_HOST", "envhost")
	os.Setenv("DB_PORT", "5434")
	os.Setenv("DB_USER", "envuser")
	os.Setenv("DB_PASSWORD", "envpass")
	os.Setenv("DB_NAME", "envdb")
	os.Setenv("DB_SSLMODE", "require")
	os.Setenv("SIDECAR_HOST", "envsidecar")
	os.Setenv("SIDECAR_PORT", "4003")
	os.Setenv("SIDECAR_CHANNEL", "envchannel")
	os.Setenv("SIDECAR_START_BLOCK", "50")
	os.Setenv("SIDECAR_END_BLOCK", "500")
	os.Setenv("RAW_CHANNEL_SIZE", "400")
	os.Setenv("PROCESS_CHANNEL_SIZE", "400")
	os.Setenv("RECEIVER_CHANNEL_SIZE", "400")
	os.Setenv("PROCESS_WORKERS", "30")
	os.Setenv("WRITE_WORKERS", "30")
	os.Setenv("HTTP_ADDR", ":8082")
	os.Setenv("GRPC_ADDR", ":9092")
	os.Setenv("HTTP_SHUTDOWN_TIMEOUT_SEC", "25")
	os.Setenv("WRITER_WAIT_TIMEOUT_SEC", "35")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "envhost", cfg.DB.Host)
	assert.Equal(t, 5434, cfg.DB.Port)
	assert.Equal(t, "envuser", cfg.DB.User)
	assert.Equal(t, "envpass", cfg.DB.Password)
	assert.Equal(t, "envdb", cfg.DB.DBName)
	assert.Equal(t, "require", cfg.DB.SSLMode)
	assert.Equal(t, "envsidecar", cfg.Sidecar.Host)
	assert.Equal(t, 4003, cfg.Sidecar.Port)
	assert.Equal(t, "envchannel", cfg.Sidecar.ChannelID)
	assert.Equal(t, uint64(50), cfg.Sidecar.StartBlk)
	assert.Equal(t, uint64(500), cfg.Sidecar.EndBlk)
	assert.Equal(t, 400, cfg.Buffer.RawChannelSize)
	assert.Equal(t, 400, cfg.Buffer.ProcessChannelSize)
	assert.Equal(t, 400, cfg.Buffer.ReceiverChannelSize)
	assert.Equal(t, 30, cfg.Workers.ProcessorCount)
	assert.Equal(t, 30, cfg.Workers.WriterCount)
	assert.Equal(t, ":8082", cfg.Server.HTTPAddr)
	assert.Equal(t, ":9092", cfg.Server.GRPCAddr)
	assert.Equal(t, 25, cfg.Server.ShutdownTimeoutSec)
	assert.Equal(t, 35, cfg.Server.WriterWaitTimeoutSec)
}

func TestLoadWithDefaults(t *testing.T) {
	// Save original env and restore after test
	originalEnv := os.Environ()
	defer func() {
		os.Clearenv()
		for _, e := range originalEnv {
			kv := splitEnv(e)
			if len(kv) == 2 {
				os.Setenv(kv[0], kv[1])
			}
		}
	}()

	os.Clearenv()

	cfg, err := Load()
	require.NoError(t, err)

	// Verify defaults
	assert.Equal(t, "localhost", cfg.DB.Host)
	assert.Equal(t, 5432, cfg.DB.Port)
	assert.Equal(t, "postgres", cfg.DB.User)
	assert.Equal(t, "postgres", cfg.DB.Password)
	assert.Equal(t, "explorer", cfg.DB.DBName)
	assert.Equal(t, "disable", cfg.DB.SSLMode)
	assert.Equal(t, "localhost", cfg.Sidecar.Host)
	assert.Equal(t, 4001, cfg.Sidecar.Port)
	assert.Equal(t, "mychannel", cfg.Sidecar.ChannelID)
	assert.Equal(t, uint64(0), cfg.Sidecar.StartBlk)
	assert.Equal(t, ^uint64(0), cfg.Sidecar.EndBlk)
	assert.Equal(t, 200, cfg.Buffer.RawChannelSize)
	assert.Equal(t, 200, cfg.Buffer.ProcessChannelSize)
	assert.Equal(t, 200, cfg.Buffer.ReceiverChannelSize)
	assert.Equal(t, 10, cfg.Workers.ProcessorCount)
	assert.Equal(t, 10, cfg.Workers.WriterCount)
	assert.Equal(t, ":8080", cfg.Server.HTTPAddr)
	assert.Equal(t, ":9090", cfg.Server.GRPCAddr)
	assert.Equal(t, 10, cfg.Server.ShutdownTimeoutSec)
	assert.Equal(t, 15, cfg.Server.WriterWaitTimeoutSec)
}

func TestGetEnv(t *testing.T) {
	os.Setenv("TEST_VAR", "testvalue")
	defer os.Unsetenv("TEST_VAR")

	assert.Equal(t, "testvalue", getEnv("TEST_VAR", "default"))
	assert.Equal(t, "default", getEnv("NONEXISTENT_VAR", "default"))
}

func TestGetInt(t *testing.T) {
	os.Setenv("TEST_INT", "42")
	defer os.Unsetenv("TEST_INT")

	assert.Equal(t, 42, getInt("TEST_INT", 10))
	assert.Equal(t, 10, getInt("NONEXISTENT_INT", 10))

	os.Setenv("TEST_INVALID_INT", "notanumber")
	defer os.Unsetenv("TEST_INVALID_INT")
	assert.Equal(t, 10, getInt("TEST_INVALID_INT", 10))
}

func TestGetUint(t *testing.T) {
	os.Setenv("TEST_UINT", "12345")
	defer os.Unsetenv("TEST_UINT")

	assert.Equal(t, uint64(12345), getUint("TEST_UINT", 100))
	assert.Equal(t, uint64(100), getUint("NONEXISTENT_UINT", 100))

	os.Setenv("TEST_INVALID_UINT", "notanumber")
	defer os.Unsetenv("TEST_INVALID_UINT")
	assert.Equal(t, uint64(100), getUint("TEST_INVALID_UINT", 100))
}

// Helper function to split environment variable strings
func splitEnv(env string) []string {
	for i := 0; i < len(env); i++ {
		if env[i] == '=' {
			return []string{env[:i], env[i+1:]}
		}
	}
	return []string{env}
}
