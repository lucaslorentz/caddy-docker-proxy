package caddyfile

import (
	"io/ioutil"
	"regexp"
	"strings"
	"testing"

	_ "github.com/caddyserver/caddy/v2/modules/standard" // plug standard HTTP modules
	"github.com/stretchr/testify/assert"
)

func TestMarshalUnmarshal(t *testing.T) {
	const folder = "./testdata/marshal"

	// load the list of test files from the dir
	files, err := ioutil.ReadDir(folder)
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

		t.Run(filename, func(t *testing.T) {
			data, err := ioutil.ReadFile(folder + "/" + filename)
			if err != nil {
				t.Errorf("failed to read %s dir: %s", filename, err)
			}

			// replace windows newlines in the json with unix newlines
			content := winNewlines.ReplaceAllString(string(data), "\n")

			// split two Caddyfile parts
			parts := strings.Split(content, "----------\n")
			beforeCaddyfile, expectedCaddyfile := parts[0], parts[1]

			container, _ := Unmarshal([]byte(beforeCaddyfile))
			result := string(container.Marshal())

			actualCaddyfile := string(result)

			// compare the actual and expected Caddyfiles
			assert.Equal(t, expectedCaddyfile, actualCaddyfile,
				"failed to process in %s", filename)
		})
	}
}
