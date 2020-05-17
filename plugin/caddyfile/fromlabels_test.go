package caddyfile

import (
	"testing"

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

	caddyfileBlock := FromLabels(labels, nil)

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

	assert.Equal(t, expectedCaddyfile, caddyfileBlock.MarshalString())
}

func TestFromLabels_FollowAlphabeticalOrder(t *testing.T) {
	labels := map[string]string{
		"caddy":     "localhost",
		"caddy.bbb": "value",
		"caddy.aaa": "value",
	}

	caddyfileBlock := FromLabels(labels, nil)

	const expectedCaddyfile = "localhost {\n" +
		"	aaa value\n" +
		"	bbb value\n" +
		"}\n"

	assert.Equal(t, expectedCaddyfile, caddyfileBlock.MarshalString())
}

func TestFromLabels_OrderDirectivesWithPrefix(t *testing.T) {
	labels := map[string]string{
		"caddy":       "localhost",
		"caddy.1_bbb": "value",
		"caddy.2_aaa": "value",
	}

	caddyfileBlock := FromLabels(labels, nil)

	const expectedCaddyfile = "localhost {\n" +
		"	bbb value\n" +
		"	aaa value\n" +
		"}\n"

	assert.Equal(t, expectedCaddyfile, caddyfileBlock.MarshalString())
}

func TestFromLabels_SeparateDirectivesWithSuffix(t *testing.T) {
	labels := map[string]string{
		"caddy":           "localhost",
		"caddy.group_1.a": "value",
		"caddy.group_2.b": "value",
	}

	caddyfileBlock := FromLabels(labels, nil)

	const expectedCaddyfile = "localhost {\n" +
		"	group {\n" +
		"		a value\n" +
		"	}\n" +
		"	group {\n" +
		"		b value\n" +
		"	}\n" +
		"}\n"

	assert.Equal(t, expectedCaddyfile, caddyfileBlock.MarshalString())
}

func TestFromLabels_EmptyValuesUsingTemplates(t *testing.T) {
	labels := map[string]string{
		"caddy": "localhost",
		// This is actually wrong, and template engine returns an error that we ignore
		// Let's keep like that for now to be backwards compatible
		// Such feature is usefull because some docker UI doesn't allow empty labels
		"caddy.key": "{{nil}}",
	}

	caddyfileBlock := FromLabels(labels, nil)

	const expectedCaddyfile = "localhost {\n" +
		"	key\n" +
		"}\n"

	assert.Equal(t, expectedCaddyfile, caddyfileBlock.MarshalString())
}

func TestFromLabels_GlobalDirectives(t *testing.T) {
	labels := map[string]string{
		"caddy.key": "value",
	}

	caddyfileBlock := FromLabels(labels, nil)

	const expectedCaddyfile = "key value\n"

	assert.Equal(t, expectedCaddyfile, caddyfileBlock.MarshalString())
}
