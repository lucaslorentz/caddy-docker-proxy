package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLabelsToCaddyfile_MinimumSpecialLabels(t *testing.T) {
	labels := map[string]string{
		"caddy":               "service.testdomain.com",
		"caddy.reverse_proxy": "{{upstreams}}",
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

func TestLabelsToCaddyfile_WithGroups(t *testing.T) {
	labels := map[string]string{
		"caddy":               "service.testdomain.com",
		"caddy.reverse_proxy": "{{upstreams https 5000}}",
		"caddy.rewrite":       "* /api{path}",
	}

	caddyfileBlock, err := labelsToCaddyfile(labels, nil, func() ([]string, error) {
		return []string{"target"}, nil
	})

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"	reverse_proxy https://target:5000\n" +
		"	rewrite * /api{path}\n" +
		"}\n"

	assert.NoError(t, err)
	assert.Equal(t, expectedCaddyfile, caddyfileBlock.MarshalString())
}

func TestLabelsToCaddyfile_AllSpecialLabels(t *testing.T) {
	labels := map[string]string{
		"caddy":                       "service.testdomain.com",
		"caddy.route":                 "/path/*",
		"caddy.route.0_uri":           "strip_prefix /path",
		"caddy.route.1_rewrite":       "* /api{path}",
		"caddy.route.2_reverse_proxy": "{{upstreams https 5000}}",
	}

	caddyfileBlock, err := labelsToCaddyfile(labels, nil, func() ([]string, error) {
		return []string{"target"}, nil
	})

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"	route /path/* {\n" +
		"		uri strip_prefix /path\n" +
		"		rewrite * /api{path}\n" +
		"		reverse_proxy https://target:5000\n" +
		"	}\n" +
		"}\n"

	assert.NoError(t, err)
	assert.Equal(t, expectedCaddyfile, caddyfileBlock.MarshalString())
}

func TestLabelsToCaddyfile_MultipleConfigs(t *testing.T) {
	labels := map[string]string{
		"caddy_0":               "service1.testdomain.com",
		"caddy_0.reverse_proxy": "{{upstreams 5000}}",
		"caddy_0.rewrite":       "* /api{path}",
		"caddy_0.tls.dns":       "route53",
		"caddy_1":               "service2.testdomain.com",
		"caddy_1.reverse_proxy": "{{upstreams 5001}}",
		"caddy_1.tls.dns":       "route53",
	}

	caddyfileBlock, err := labelsToCaddyfile(labels, nil, func() ([]string, error) {
		return []string{"target"}, nil
	})

	const expectedCaddyfile = "service1.testdomain.com {\n" +
		"	reverse_proxy target:5000\n" +
		"	rewrite * /api{path}\n" +
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
		"caddy":               "a.testdomain.com b.testdomain.com",
		"caddy.reverse_proxy": "{{upstreams}}",
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
		"caddy":                 "testdomain.com",
		"caddy.reverse_proxy":   "something",
		"caddy.reverse_proxy_1": "/api/* external-api",
	}

	caddyfileBlock, err := labelsToCaddyfile(labels, nil, func() ([]string, error) {
		return []string{"target"}, nil
	})

	const expectedCaddyfile = "testdomain.com {\n" +
		"	reverse_proxy /api/* external-api\n" +
		"	reverse_proxy something\n" +
		"}\n"

	assert.NoError(t, err)
	assert.Equal(t, expectedCaddyfile, caddyfileBlock.MarshalString())
}

func TestLabelsToCaddyfile_ReverseProxyDirectivesAreMovedIntoRoute(t *testing.T) {
	labels := map[string]string{
		"caddy":                       "service.testdomain.com",
		"caddy.route":                 "/path/*",
		"caddy.route.0_uri":           "strip_prefix /path",
		"caddy.route.1_reverse_proxy": "{{upstreams}}",
		"caddy.route.1_reverse_proxy.health_path": "/health",
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

func TestLabelsToCaddyfile_InvalidTemplate(t *testing.T) {
	labels := map[string]string{
		"caddy":               "service.testdomain.com",
		"caddy.reverse_proxy": "{{invalid}}",
	}

	caddyfileBlock, err := labelsToCaddyfile(labels, nil, func() ([]string, error) {
		return []string{"target"}, nil
	})

	assert.Error(t, err, `template: :1: function "invalid" not defined`)
	assert.Nil(t, caddyfileBlock)
}
