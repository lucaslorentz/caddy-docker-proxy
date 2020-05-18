package caddyfile

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestFromLabels_Grouping(t *testing.T) {
	labels := map[string]string{
		"caddy":               "localhost",
		"caddy.group.a":       "value-a",
		"caddy.group.b":       "value-b",
		"caddy.group.group.a": "value-a",
		"caddy.group.group.b": "value-b",
	}

	caddyfileBlock, err := FromLabels(labels, nil, template.FuncMap{})

	const expectedCaddyfile = "localhost {\n" +
		"	group {\n" +
		"		a value-a\n" +
		"		b value-b\n" +
		"		group {\n" +
		"			a value-a\n" +
		"			b value-b\n" +
		"		}\n" +
		"	}\n" +
		"}\n"

	assert.NoError(t, err)
	assert.Equal(t, expectedCaddyfile, caddyfileBlock.MarshalString())
}

func TestFromLabels_FollowAlphabeticalOrder(t *testing.T) {
	labels := map[string]string{
		"caddy":     "localhost",
		"caddy.bbb": "value",
		"caddy.aaa": "value",
	}

	caddyfileBlock, err := FromLabels(labels, nil, template.FuncMap{})

	const expectedCaddyfile = "localhost {\n" +
		"	aaa value\n" +
		"	bbb value\n" +
		"}\n"

	assert.NoError(t, err)
	assert.Equal(t, expectedCaddyfile, caddyfileBlock.MarshalString())
}

func TestFromLabels_OrderDirectivesWithPrefix(t *testing.T) {
	labels := map[string]string{
		"caddy":       "localhost",
		"caddy.1_bbb": "value",
		"caddy.2_aaa": "value",
	}

	caddyfileBlock, err := FromLabels(labels, nil, template.FuncMap{})

	const expectedCaddyfile = "localhost {\n" +
		"	bbb value\n" +
		"	aaa value\n" +
		"}\n"

	assert.NoError(t, err)
	assert.Equal(t, expectedCaddyfile, caddyfileBlock.MarshalString())
}

func TestFromLabels_SeparateDirectivesWithSuffix(t *testing.T) {
	labels := map[string]string{
		"caddy":           "localhost",
		"caddy.group_1.a": "value",
		"caddy.group_2.b": "value",
	}

	caddyfileBlock, err := FromLabels(labels, nil, template.FuncMap{})

	const expectedCaddyfile = "localhost {\n" +
		"	group {\n" +
		"		a value\n" +
		"	}\n" +
		"	group {\n" +
		"		b value\n" +
		"	}\n" +
		"}\n"

	assert.NoError(t, err)
	assert.Equal(t, expectedCaddyfile, caddyfileBlock.MarshalString())
}

func TestFromLabels_TemplatesEmptyValues(t *testing.T) {
	labels := map[string]string{
		"caddy":     "localhost",
		"caddy.key": `{{""}}`,
	}

	caddyfileBlock, err := FromLabels(labels, nil, template.FuncMap{})

	const expectedCaddyfile = "localhost {\n" +
		"	key\n" +
		"}\n"

	assert.NoError(t, err)
	assert.Equal(t, expectedCaddyfile, caddyfileBlock.MarshalString())
}

func TestFromLabels_TemplateError(t *testing.T) {
	labels := map[string]string{
		"caddy.key": "{{invalid}}",
	}

	caddyfileBlock, err := FromLabels(labels, nil, template.FuncMap{})

	const expectedCaddyfile = ""

	assert.Error(t, err)
	assert.Nil(t, caddyfileBlock)
}

func TestFromLabels_GlobalDirectives(t *testing.T) {
	labels := map[string]string{
		"caddy.key": "value",
	}

	caddyfileBlock, err := FromLabels(labels, nil, template.FuncMap{})

	const expectedCaddyfile = "key value\n"

	assert.NoError(t, err)
	assert.Equal(t, expectedCaddyfile, caddyfileBlock.MarshalString())
}

func TestFromLabels_SnippetsComesFirst(t *testing.T) {
	labels := map[string]string{
		"caddy_0":        "aaa.com",
		"caddy_0.import": "my-snippet",
		"caddy_1":        "(my-snippet)",
		"caddy_1.tls":    "internal",
	}

	caddyfileBlock, err := FromLabels(labels, nil, template.FuncMap{})

	const expectedCaddyfile = "(my-snippet) {\n" +
		"	tls internal\n" +
		"}\n" +
		"aaa.com {\n" +
		"	import my-snippet\n" +
		"}\n"

	assert.NoError(t, err)
	assert.Equal(t, expectedCaddyfile, caddyfileBlock.MarshalString())
}

func TestFromLabels_MatchersComesFirst(t *testing.T) {
	labels := map[string]string{
		"caddy":               "localhost",
		"caddy.@matcher.path": "/path1 /path2",
		"caddy.respond":       "@matcher 200",
	}

	caddyfileBlock, err := FromLabels(labels, nil, template.FuncMap{})

	const expectedCaddyfile = "localhost {\n" +
		"	@matcher {\n" +
		"		path /path1 /path2\n" +
		"	}\n" +
		"	respond @matcher 200\n" +
		"}\n"

	assert.NoError(t, err)
	assert.Equal(t, expectedCaddyfile, caddyfileBlock.MarshalString())
}
