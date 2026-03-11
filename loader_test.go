package caddydockerproxy

import (
	"encoding/json"
	"testing"

	"github.com/caddyserver/caddy/v2"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddAdminListenAddsFallbackWhenMissing(t *testing.T) {
	input, err := json.Marshal(&caddy.Config{})
	require.NoError(t, err)

	output, err := addAdminListen(input, "tcp/localhost:2019")
	require.NoError(t, err)

	result := &caddy.Config{}
	err = json.Unmarshal(output, result)
	require.NoError(t, err)
	require.NotNil(t, result.Admin)
	assert.Equal(t, "tcp/localhost:2019", result.Admin.Listen)
}

func TestAddAdminListenKeepsExistingAdminListen(t *testing.T) {
	input, err := json.Marshal(&caddy.Config{
		Admin: &caddy.AdminConfig{
			Listen: "tcp/0.0.0.0:2019",
		},
	})
	require.NoError(t, err)

	output, err := addAdminListen(input, "tcp/localhost:2019")
	require.NoError(t, err)

	result := &caddy.Config{}
	err = json.Unmarshal(output, result)
	require.NoError(t, err)
	require.NotNil(t, result.Admin)
	assert.Equal(t, "tcp/0.0.0.0:2019", result.Admin.Listen)
}

func TestAddAdminListenOverridesAdminOff(t *testing.T) {
	input, err := json.Marshal(&caddy.Config{
		Admin: &caddy.AdminConfig{
			Disabled: true,
		},
	})
	require.NoError(t, err)

	output, err := addAdminListen(input, "tcp/localhost:2019")
	require.NoError(t, err)

	result := &caddy.Config{}
	err = json.Unmarshal(output, result)
	require.NoError(t, err)
	require.NotNil(t, result.Admin)
	assert.False(t, result.Admin.Disabled)
	assert.Equal(t, "tcp/localhost:2019", result.Admin.Listen)
}

func TestGetServerAdminListen(t *testing.T) {
	assert.Equal(t, "tcp/10.0.0.2:2019", getServerAdminListen(&config.Options{}, "10.0.0.2"))
	assert.Equal(t, "tcp/0.0.0.0:2019", getServerAdminListen(&config.Options{AdminListen: "tcp/0.0.0.0:2019"}, "10.0.0.2"))
}
