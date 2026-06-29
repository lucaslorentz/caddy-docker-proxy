package caddydockerproxy

import (
	"encoding/json"
	"testing"

	"github.com/caddyserver/caddy/v2"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetServerAdminListen(t *testing.T) {
	assert.Equal(t, "tcp/10.0.0.2:2019", getServerAdminListen(&config.Options{}, "10.0.0.2"))
	assert.Equal(t, "tcp/0.0.0.0:2019", getServerAdminListen(&config.Options{AdminListen: "tcp/0.0.0.0:2019"}, "10.0.0.2"))
}

func unmarshalConfig(t *testing.T, data []byte) *caddy.Config {
	t.Helper()
	config := &caddy.Config{}
	require.NoError(t, json.Unmarshal(data, config))
	return config
}

func TestPrepareServerConfig(t *testing.T) {
	empty, err := json.Marshal(&caddy.Config{})
	require.NoError(t, err)

	loggingERROR := buildCaddyLoggingConfig(&config.Options{Mode: config.Standalone, LogLevel: "ERROR"})

	newLoader := func(configJSON []byte, logging *caddy.Logging) *DockerLoader {
		return &DockerLoader{options: &config.Options{}, lastJSONConfig: configJSON, caddyLogging: logging}
	}

	t.Run("adds admin listen fallback when missing", func(t *testing.T) {
		out, err := newLoader(empty, nil).prepareServerConfig("10.0.0.2")
		require.NoError(t, err)
		assert.Equal(t, "tcp/10.0.0.2:2019", unmarshalConfig(t, out).Admin.Listen)
	})

	t.Run("keeps an explicit admin listen", func(t *testing.T) {
		in, err := json.Marshal(&caddy.Config{Admin: &caddy.AdminConfig{Listen: "tcp/0.0.0.0:2019"}})
		require.NoError(t, err)
		out, err := newLoader(in, nil).prepareServerConfig("10.0.0.2")
		require.NoError(t, err)
		assert.Equal(t, "tcp/0.0.0.0:2019", unmarshalConfig(t, out).Admin.Listen)
	})

	t.Run("overrides admin off for remote servers", func(t *testing.T) {
		in, err := json.Marshal(&caddy.Config{Admin: &caddy.AdminConfig{Disabled: true}})
		require.NoError(t, err)
		out, err := newLoader(in, nil).prepareServerConfig("10.0.0.2")
		require.NoError(t, err)
		result := unmarshalConfig(t, out)
		assert.False(t, result.Admin.Disabled)
		assert.Equal(t, "tcp/10.0.0.2:2019", result.Admin.Listen)
	})

	t.Run("keeps admin off for local server", func(t *testing.T) {
		in, err := json.Marshal(&caddy.Config{Admin: &caddy.AdminConfig{Disabled: true}})
		require.NoError(t, err)
		out, err := newLoader(in, nil).prepareServerConfig("localhost")
		require.NoError(t, err)
		result := unmarshalConfig(t, out)
		assert.True(t, result.Admin.Disabled)
		assert.Empty(t, result.Admin.Listen)
	})

	t.Run("disables admin fallback for local server", func(t *testing.T) {
		out, err := newLoader(empty, nil).prepareServerConfig("localhost")
		require.NoError(t, err)
		result := unmarshalConfig(t, out)
		assert.True(t, result.Admin.Disabled)
		assert.Empty(t, result.Admin.Listen)
	})

	t.Run("injects logging on the local Caddy", func(t *testing.T) {
		out, err := newLoader(empty, loggingERROR).prepareServerConfig("localhost")
		require.NoError(t, err)
		result := unmarshalConfig(t, out)
		require.NotNil(t, result.Logging)
		assert.Equal(t, "ERROR", result.Logging.Logs["default"].Level)
	})

	t.Run("does not inject logging on remote servers", func(t *testing.T) {
		out, err := newLoader(empty, loggingERROR).prepareServerConfig("10.0.0.2")
		require.NoError(t, err)
		assert.Nil(t, unmarshalConfig(t, out).Logging)
	})

	t.Run("no logging configured leaves logging unset", func(t *testing.T) {
		out, err := newLoader(empty, nil).prepareServerConfig("localhost")
		require.NoError(t, err)
		assert.Nil(t, unmarshalConfig(t, out).Logging)
	})

	t.Run("respects logging already in the config", func(t *testing.T) {
		in, err := json.Marshal(&caddy.Config{
			Logging: &caddy.Logging{Logs: map[string]*caddy.CustomLog{"default": {BaseLog: caddy.BaseLog{Level: "WARN"}}}},
		})
		require.NoError(t, err)
		out, err := newLoader(in, loggingERROR).prepareServerConfig("localhost")
		require.NoError(t, err)
		assert.Equal(t, "WARN", unmarshalConfig(t, out).Logging.Logs["default"].Level)
	})
}
