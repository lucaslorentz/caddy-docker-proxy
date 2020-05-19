package caddyfile

import (
	"io/ioutil"
	"regexp"
	"strings"
	"testing"

	_ "github.com/caddyserver/caddy/v2/modules/standard" // plug standard HTTP modules

	"github.com/stretchr/testify/assert"
)

func TestProcessCaddyfile(t *testing.T) {
	// load the list of test files from the dir
	files, err := ioutil.ReadDir("./testdata/process")
	if err != nil {
		t.Errorf("failed to read process dir: %s", err)
	}

	// prep a regexp to fix strings on windows
	winNewlines := regexp.MustCompile(`\r?\n`)

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		// read the test file
		filename := f.Name()
		data, err := ioutil.ReadFile("./testdata/process/" + filename)
		if err != nil {
			t.Errorf("failed to read %s dir: %s", filename, err)
		}

		// split two Caddyfile parts
		parts := strings.Split(string(data), "----------")
		beforeCaddyfile, expectedCaddyfile := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])

		// replace windows newlines in the json with unix newlines
		expectedCaddyfile = winNewlines.ReplaceAllString(expectedCaddyfile, "\n")

		// process the Caddyfile
		result, _ := Process([]byte(beforeCaddyfile))

		actualCaddyfile := strings.TrimSpace(string(result))

		// compare the actual and expected Caddyfiles
		assert.Equal(t, expectedCaddyfile, actualCaddyfile,
			"failed to process in %s, \nExpected:\n%s\nActual:\n%s",
			filename, expectedCaddyfile, actualCaddyfile)
	}
}
