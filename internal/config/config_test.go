package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")
	require.NoError(t, err)

	assert.Equal(t, "8080", cfg.Server.Port)
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 25, cfg.Database.MaxOpenConns)
	assert.Equal(t, 100, cfg.RateLimit.RequestsPerMinute)
}

func TestLoad_FromFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	content := `
server:
  port: "9090"
database:
  host: "dbhost"
  port: "3307"
  user: "admin"
  password: "secret"
  dbname: "testdb"
  max_open_conns: 50
  max_idle_conns: 20
  conn_max_lifetime: 600
redis:
  addr: "redishost:6380"
  password: "redispass"
  db: 1
  pool_size: 20
jwt:
  secret: "my-secret"
  expiration: 48
rate_limit:
  requests_per_minute: 200
`
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	cfg, err := Load(tmpFile.Name())
	require.NoError(t, err)

	assert.Equal(t, "9090", cfg.Server.Port)
	assert.Equal(t, "dbhost", cfg.Database.Host)
	assert.Equal(t, 50, cfg.Database.MaxOpenConns)
	assert.Equal(t, "redishost:6380", cfg.Redis.Addr)
	assert.Equal(t, "my-secret", cfg.JWT.Secret)
	assert.Equal(t, 48, cfg.JWT.Expiration)
	assert.Equal(t, 200, cfg.RateLimit.RequestsPerMinute)
}

func TestLoad_EnvOverrides(t *testing.T) {
	os.Setenv("SERVER_PORT", "7777")
	os.Setenv("DB_HOST", "envhost")
	os.Setenv("JWT_SECRET", "env-secret")
	defer func() {
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("DB_HOST")
		os.Unsetenv("JWT_SECRET")
	}()

	cfg, err := Load("/nonexistent/path/config.yaml")
	require.NoError(t, err)

	assert.Equal(t, "7777", cfg.Server.Port)
	assert.Equal(t, "envhost", cfg.Database.Host)
	assert.Equal(t, "env-secret", cfg.JWT.Secret)
}

func TestDSN(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "localhost",
		Port:     "3306",
		User:     "root",
		Password: "pass",
		DBName:   "mydb",
	}
	expected := "root:pass@tcp(localhost:3306)/mydb?parseTime=true&charset=utf8mb4"
	assert.Equal(t, expected, cfg.DSN())
}
