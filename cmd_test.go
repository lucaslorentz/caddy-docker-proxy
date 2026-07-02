package caddydockerproxy

import (
	"testing"

	"github.com/lucaslorentz/caddy-docker-proxy/v2/config"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeAdminListen(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty",
			input:    "",
			expected: "",
		},
		{
			name:     "trim and add tcp prefix",
			input:    " 0.0.0.0:2019 ",
			expected: "tcp/0.0.0.0:2019",
		},
		{
			name:     "keep prefixed listen value",
			input:    "tcp/0.0.0.0:2019",
			expected: "tcp/0.0.0.0:2019",
		},
		{
			name:     "keep unix listen value",
			input:    "unix//run/caddy-admin.sock",
			expected: "unix//run/caddy-admin.sock",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, normalizeAdminListen(testCase.input))
		})
	}
}

func TestParseAdminEnv(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectListen   string
		expectDisabled bool
	}{
		{name: "off disables admin", input: "off", expectListen: "", expectDisabled: true},
		{name: "off is case-insensitive", input: "OFF", expectListen: "", expectDisabled: true},
		{name: "off is trimmed", input: "  off  ", expectListen: "", expectDisabled: true},
		{name: "address without scheme gets tcp prefix", input: "0.0.0.0:2019", expectListen: "tcp/0.0.0.0:2019"},
		{name: "prefixed address is kept", input: "tcp/localhost:2019", expectListen: "tcp/localhost:2019"},
		{name: "unix socket is kept", input: "unix//run/caddy-admin.sock", expectListen: "unix//run/caddy-admin.sock"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			listen, disabled := parseAdminEnv(testCase.input)
			assert.Equal(t, testCase.expectListen, listen)
			assert.Equal(t, testCase.expectDisabled, disabled)
		})
	}
}

func TestGetAdminListenPrefersConfiguredListen(t *testing.T) {
	options := &config.Options{
		AdminListen: "tcp/0.0.0.0:2019",
	}

	assert.Equal(t, "tcp/0.0.0.0:2019", getAdminListen(options))
}

func TestBuildCaddyLoggingConfig(t *testing.T) {
	t.Run("empty default log when nothing customized (Caddy defaults apply)", func(t *testing.T) {
		logging := buildCaddyLoggingConfig(&config.Options{Mode: config.Standalone})
		assert.NotNil(t, logging)
		dl := logging.Logs["default"]
		assert.Empty(t, dl.Level)
		assert.Nil(t, dl.EncoderRaw)
		assert.Empty(t, dl.Exclude)
	})

	t.Run("sets default log level", func(t *testing.T) {
		logging := buildCaddyLoggingConfig(&config.Options{Mode: config.Standalone, LogLevel: "error"})
		assert.NotNil(t, logging)
		assert.Equal(t, "ERROR", logging.Logs["default"].Level)
		assert.Nil(t, logging.Logs["default"].EncoderRaw)
		assert.Empty(t, logging.Logs["default"].Exclude)
	})

	t.Run("sets console encoder", func(t *testing.T) {
		logging := buildCaddyLoggingConfig(&config.Options{Mode: config.Standalone, LogFormat: "console"})
		assert.NotNil(t, logging)
		assert.JSONEq(t, `{"format":"console"}`, string(logging.Logs["default"].EncoderRaw))
	})

	t.Run("sets json encoder and level together", func(t *testing.T) {
		logging := buildCaddyLoggingConfig(&config.Options{Mode: config.Standalone, LogLevel: "WARN", LogFormat: "json"})
		assert.NotNil(t, logging)
		assert.Equal(t, "WARN", logging.Logs["default"].Level)
		assert.JSONEq(t, `{"format":"json"}`, string(logging.Logs["default"].EncoderRaw))
	})

	t.Run("controller-only excludes the admin logger", func(t *testing.T) {
		logging := buildCaddyLoggingConfig(&config.Options{Mode: config.Controller, LogLevel: "error"})
		assert.NotNil(t, logging)
		assert.Contains(t, logging.Logs["default"].Exclude, "admin")
	})

	t.Run("admin disabled excludes the admin logger", func(t *testing.T) {
		logging := buildCaddyLoggingConfig(&config.Options{Mode: config.Standalone, AdminDisabled: true})
		assert.NotNil(t, logging)
		assert.Contains(t, logging.Logs["default"].Exclude, "admin")
	})
}

func TestBuildCaddyRunConfig(t *testing.T) {
	t.Run("server/standalone uses the default admin listen", func(t *testing.T) {
		cfg := buildCaddyRunConfig(&config.Options{Mode: config.Standalone})
		assert.NotNil(t, cfg)
		assert.NotNil(t, cfg.Admin)
		assert.False(t, cfg.Admin.Disabled)
		assert.Equal(t, "tcp/localhost:2019", cfg.Admin.Listen)
	})

	t.Run("controller-only disables admin and excludes admin logs", func(t *testing.T) {
		cfg := buildCaddyRunConfig(&config.Options{Mode: config.Controller, LogLevel: "ERROR"})
		assert.NotNil(t, cfg)
		assert.NotNil(t, cfg.Admin)
		assert.True(t, cfg.Admin.Disabled)
		assert.Equal(t, "ERROR", cfg.Logging.Logs["default"].Level)
		assert.Contains(t, cfg.Logging.Logs["default"].Exclude, "admin")
	})

	t.Run("CADDY_ADMIN=off disables admin and excludes admin logs", func(t *testing.T) {
		cfg := buildCaddyRunConfig(&config.Options{Mode: config.Standalone, AdminDisabled: true})
		assert.NotNil(t, cfg)
		assert.NotNil(t, cfg.Admin)
		assert.True(t, cfg.Admin.Disabled)
		assert.Contains(t, cfg.Logging.Logs["default"].Exclude, "admin")
	})
}

func TestBuildCaddyAdminConfig(t *testing.T) {
	t.Run("server/standalone uses the default admin listen", func(t *testing.T) {
		admin := buildCaddyAdminConfig(&config.Options{Mode: config.Standalone})
		assert.False(t, admin.Disabled)
		assert.Equal(t, "tcp/localhost:2019", admin.Listen)
	})

	t.Run("uses the CADDY_ADMIN listen when set", func(t *testing.T) {
		admin := buildCaddyAdminConfig(&config.Options{Mode: config.Standalone, AdminListen: "tcp/0.0.0.0:2019"})
		assert.False(t, admin.Disabled)
		assert.Equal(t, "tcp/0.0.0.0:2019", admin.Listen)
	})

	t.Run("controller-only disables admin", func(t *testing.T) {
		assert.True(t, buildCaddyAdminConfig(&config.Options{Mode: config.Controller}).Disabled)
	})

	t.Run("CADDY_ADMIN=off disables admin", func(t *testing.T) {
		assert.True(t, buildCaddyAdminConfig(&config.Options{Mode: config.Standalone, AdminDisabled: true}).Disabled)
	})
}
