package caddydockerproxy

import (
	"testing"

	"github.com/caddyserver/caddy/v2"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

// Verifies that the logging config produced from --log-level/--log-format is
// accepted by caddy.Run and applied to caddy.Log(). This is also the exact path
// used in controller-only mode (caddy.Run with admin disabled + logging).
func TestIntegration_CaddyFollowsLogArgs(t *testing.T) {
	cfg := &caddy.Config{
		Admin:   &caddy.AdminConfig{Disabled: true},
		Logging: buildCaddyLoggingConfig(&config.Options{LogLevel: "ERROR", LogFormat: "json"}),
	}
	require.NoError(t, caddy.Run(cfg))
	t.Cleanup(func() { _ = caddy.Stop() })

	core := caddy.Log().Core()
	assert.False(t, core.Enabled(zapcore.InfoLevel), "INFO should be filtered after --log-level=ERROR")
	assert.False(t, core.Enabled(zapcore.WarnLevel), "WARN should be filtered after --log-level=ERROR")
	assert.True(t, core.Enabled(zapcore.ErrorLevel), "ERROR should pass")
}
