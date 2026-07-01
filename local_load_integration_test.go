package caddydockerproxy

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/config"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pushLocal must surface a bad config as an error (and must not panic), so a
// poison config fails the reload instead of crashing the process.
func TestPushLocal_ReturnsErrorOnInvalidConfig(t *testing.T) {
	err := pushLocal([]byte("this is not valid json"))
	require.Error(t, err)
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// runLocalPush boots a Caddy instance with the admin API disabled, pushes a
// generated config through updateServer("localhost"), and asserts it went live
// in-process (the served port responds). Because the instance boots with the
// admin API disabled, a fall back to the HTTP push would POST to the admin
// endpoint and fail - so a live served port proves the in-process caddy.Load
// path. Returns the served app port.
func runLocalPush(t *testing.T, options *config.Options) int {
	t.Helper()
	appPort := freePort(t)

	caddyfile := fmt.Sprintf(":%d {\n\trespond \"docker-proxy-local-load\"\n}\n", appPort)
	configJSON, warn, err := caddyconfig.GetAdapter("caddyfile").Adapt([]byte(caddyfile), nil)
	require.NoError(t, err)
	require.Nil(t, warn)

	require.NoError(t, caddy.Run(&caddy.Config{Admin: &caddy.AdminConfig{Disabled: true}}))
	t.Cleanup(func() { _ = caddy.Stop() })

	loader := &DockerLoader{
		options:         options,
		lastJSONConfig:  configJSON,
		lastVersion:     1,
		serversVersions: utils.NewStringInt64CMap(),
		serversUpdating: utils.NewStringBoolCMap(),
	}

	var wg sync.WaitGroup
	wg.Add(1)
	loader.updateServer(&wg, localServer)
	wg.Wait()

	// The version is only recorded on a successful load.
	require.Equal(t, int64(1), loader.serversVersions.Get("localhost"))

	// Prove the config is live in-process by hitting the served port.
	url := fmt.Sprintf("http://127.0.0.1:%d/", appPort)
	var body string
	var status int
	require.Eventually(t, func() bool {
		resp, err := http.Get(url)
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return false
		}
		body = string(b)
		status = resp.StatusCode
		return true
	}, 5*time.Second, 50*time.Millisecond)

	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "docker-proxy-local-load", body)

	return appPort
}

// Verifies that updateServer("localhost") applies the generated config straight
// into the running Caddy via caddy.Load, with no admin API involved: booted with
// the admin API disabled and CADDY_ADMIN=off, the served config can only go live
// through the in-process load path.
func TestIntegration_LocalPushUsesCaddyLoad(t *testing.T) {
	runLocalPush(t, &config.Options{AdminDisabled: true})
}

// The admin API is enabled by default, so a local push must bring it up on the
// configured address (here via CADDY_ADMIN) - health checks and /metrics that
// rely on it keep working. See #820.
func TestIntegration_LocalPushEnablesAdmin(t *testing.T) {
	adminPort := freePort(t)
	runLocalPush(t, &config.Options{AdminListen: fmt.Sprintf("tcp/localhost:%d", adminPort)})

	adminURL := fmt.Sprintf("http://127.0.0.1:%d/config/", adminPort)
	require.Eventually(t, func() bool {
		resp, err := http.Get(adminURL)
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 5*time.Second, 50*time.Millisecond, "local load must keep the admin API enabled")
}
