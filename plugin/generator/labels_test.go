package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLabelsToCaddyfile_MinimumSpecialLabels(t *testing.T) {
	labels := map[string]string{
		"caddy.address": "service.testdomain.com",
	}

	caddyfileBlock, err := labelsToCaddyfile(labels, nil, func() ([]string, error) {
		return []string{"target"}, nil
	})

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"	reverse_proxy target\n" +
		"}\n"

	assert.NoError(t, err)
	assert.Equal(t, expectedCaddyfile, caddyfileBlock.MarshalString())
}

func TestLabelsToCaddyfile_AllSpecialLabelsExceptSourcePath(t *testing.T) {
	labels := map[string]string{
		"caddy.address":        "service.testdomain.com",
		"caddy.targetport":     "5000",
		"caddy.targetpath":     "/api",
		"caddy.targetprotocol": "https",
	}

	caddyfileBlock, err := labelsToCaddyfile(labels, nil, func() ([]string, error) {
		return []string{"target"}, nil
	})

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"	reverse_proxy https://target:5000\n" +
		"	rewrite * /api{uri}\n" +
		"}\n"

	assert.NoError(t, err)
	assert.Equal(t, expectedCaddyfile, caddyfileBlock.MarshalString())
}

func TestLabelsToCaddyfile_AllSpecialLabels(t *testing.T) {
	labels := map[string]string{
		"caddy.address":        "service.testdomain.com",
		"caddy.sourcepath":     "/path",
		"caddy.targetport":     "5000",
		"caddy.targetpath":     "/api",
		"caddy.targetprotocol": "https",
	}

	caddyfileBlock, err := labelsToCaddyfile(labels, nil, func() ([]string, error) {
		return []string{"target"}, nil
	})

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"	route /path/* {\n" +
		"		uri strip_prefix /path\n" +
		"		rewrite * /api{uri}\n" +
		"		reverse_proxy https://target:5000\n" +
		"	}\n" +
		"}\n"

	assert.NoError(t, err)
	assert.Equal(t, expectedCaddyfile, caddyfileBlock.MarshalString())
}

func TestLabelsToCaddyfile_MultipleConfigs(t *testing.T) {
	labels := map[string]string{
		"caddy_0.address":    "service1.testdomain.com",
		"caddy_0.targetport": "5000",
		"caddy_0.targetpath": "/api",
		"caddy_0.tls.dns":    "route53",
		"caddy_1.address":    "service2.testdomain.com",
		"caddy_1.targetport": "5001",
		"caddy_1.tls.dns":    "route53",
	}

	caddyfileBlock, err := labelsToCaddyfile(labels, nil, func() ([]string, error) {
		return []string{"target"}, nil
	})

	const expectedCaddyfile = "service1.testdomain.com {\n" +
		"	reverse_proxy target:5000\n" +
		"	rewrite * /api{uri}\n" +
		"	tls {\n" +
		"		dns route53\n" +
		"	}\n" +
		"}\n" +
		"service2.testdomain.com {\n" +
		"	reverse_proxy target:5001\n" +
		"	tls {\n" +
		"		dns route53\n" +
		"	}\n" +
		"}\n"

	assert.NoError(t, err)
	assert.Equal(t, expectedCaddyfile, caddyfileBlock.MarshalString())
}

func TestLabelsToCaddyfile_MultipleAddresses(t *testing.T) {
	labels := map[string]string{
		"caddy.address": "a.testdomain.com b.testdomain.com",
	}

	caddyfileBlock, err := labelsToCaddyfile(labels, nil, func() ([]string, error) {
		return []string{"target"}, nil
	})

	const expectedCaddyfile = "a.testdomain.com b.testdomain.com {\n" +
		"	reverse_proxy target\n" +
		"}\n"

	assert.NoError(t, err)
	assert.Equal(t, expectedCaddyfile, caddyfileBlock.MarshalString())
}

func TestLabelsToCaddyfile_DoesntOverrideExistingProxy(t *testing.T) {
	labels := map[string]string{
		"caddy.address":         "testdomain.com",
		"caddy.reverse_proxy":   "something",
		"caddy.reverse_proxy_1": "/api/* external-api",
	}

	caddyfileBlock, err := labelsToCaddyfile(labels, nil, func() ([]string, error) {
		return []string{"target"}, nil
	})

	const expectedCaddyfile = "testdomain.com {\n" +
		"	reverse_proxy something\n" +
		"	reverse_proxy /api/* external-api\n" +
		"}\n"

	assert.NoError(t, err)
	assert.Equal(t, expectedCaddyfile, caddyfileBlock.MarshalString())
}

func TestLabelsToCaddyfile_ReverseProxyDirectivesAreMovedIntoRoute(t *testing.T) {
	labels := map[string]string{
		"caddy.address":                   "service.testdomain.com",
		"caddy.sourcepath":                "/path",
		"caddy.reverse_proxy.health_path": "/health",
	}

	caddyfileBlock, err := labelsToCaddyfile(labels, nil, func() ([]string, error) {
		return []string{"target"}, nil
	})

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"	route /path/* {\n" +
		"		uri strip_prefix /path\n" +
		"		reverse_proxy target {\n" +
		"			health_path /health\n" +
		"		}\n" +
		"	}\n" +
		"}\n"

	assert.NoError(t, err)
	assert.Equal(t, expectedCaddyfile, caddyfileBlock.MarshalString())
}
