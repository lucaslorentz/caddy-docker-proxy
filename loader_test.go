package caddydockerproxy

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
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

func TestAdminAPIEndpoint(t *testing.T) {
	t.Run("uses default TCP admin endpoint", func(t *testing.T) {
		loader := &DockerLoader{options: &config.Options{}}

		client, url, cleanup, err := loader.adminAPIEndpoint("10.0.0.2")

		require.NoError(t, err)
		assert.Same(t, http.DefaultClient, client)
		assert.Equal(t, "http://10.0.0.2:2019/load", url)
		assert.Nil(t, cleanup)
	})

	t.Run("uses explicit TCP port with controlled server for wildcard host", func(t *testing.T) {
		loader := &DockerLoader{options: &config.Options{AdminListen: "tcp/0.0.0.0:8080"}}

		client, url, cleanup, err := loader.adminAPIEndpoint("10.0.0.2")

		require.NoError(t, err)
		assert.Same(t, http.DefaultClient, client)
		assert.Equal(t, "http://10.0.0.2:8080/load", url)
		assert.Nil(t, cleanup)
	})

	t.Run("uses explicit TCP host and port", func(t *testing.T) {
		loader := &DockerLoader{options: &config.Options{AdminListen: "tcp/127.0.0.1:8080"}}

		client, url, cleanup, err := loader.adminAPIEndpoint("10.0.0.2")

		require.NoError(t, err)
		assert.Same(t, http.DefaultClient, client)
		assert.Equal(t, "http://127.0.0.1:8080/load", url)
		assert.Nil(t, cleanup)
	})

	t.Run("uses Unix socket admin endpoint", func(t *testing.T) {
		socketPath := t.TempDir() + "/caddy-admin.sock"
		listener, err := net.Listen("unix", socketPath)
		require.NoError(t, err)

		requests := make(chan *http.Request, 1)
		server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.WriteHeader(http.StatusOK)
		})}
		done := make(chan error, 1)
		go func() {
			done <- server.Serve(listener)
		}()
		t.Cleanup(func() {
			require.NoError(t, server.Close())
			err := <-done
			if err != nil && err != http.ErrServerClosed {
				t.Errorf("server failed: %v", err)
			}
		})

		loader := &DockerLoader{options: &config.Options{AdminListen: "unix/" + socketPath + "|0200"}}

		client, url, cleanup, err := loader.adminAPIEndpoint("localhost")
		require.NoError(t, err)
		require.NotNil(t, cleanup)
		defer cleanup()
		assert.Equal(t, "http://127.0.0.1/load", url)

		resp, err := client.Post(url, "application/json", strings.NewReader(`{}`))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "/load", (<-requests).URL.Path)
	})

	t.Run("rejects invalid admin listen", func(t *testing.T) {
		loader := &DockerLoader{options: &config.Options{AdminListen: "tcp/localhost:not-a-port"}}

		_, _, _, err := loader.adminAPIEndpoint("localhost")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid admin listen address")
	})

	t.Run("rejects admin listen port ranges", func(t *testing.T) {
		loader := &DockerLoader{options: &config.Options{AdminListen: "tcp/localhost:2019-2020"}}

		_, _, _, err := loader.adminAPIEndpoint("localhost")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "must resolve to a single endpoint")
	})
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

	t.Run("overrides admin off", func(t *testing.T) {
		in, err := json.Marshal(&caddy.Config{Admin: &caddy.AdminConfig{Disabled: true}})
		require.NoError(t, err)
		out, err := newLoader(in, nil).prepareServerConfig("localhost")
		require.NoError(t, err)
		result := unmarshalConfig(t, out)
		assert.False(t, result.Admin.Disabled)
		assert.Equal(t, "tcp/localhost:2019", result.Admin.Listen)
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
