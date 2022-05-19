package caddyfile

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestLabelsToCaddyfile(t *testing.T) {
	// load the list of test files from the dir
	files, err := ioutil.ReadDir("./testdata/labels")
	if err != nil {
		t.Errorf("failed to read labels dir: %s", err)
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
			data, err := ioutil.ReadFile("./testdata/labels/" + filename)
			if err != nil {
				t.Errorf("failed to read %s dir: %s", filename, err)
			}

			// split the labels (first) and Caddyfile (second) parts
			parts := strings.Split(string(data), "----------")
			labelsString, expectedCaddyfile := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])

			// parse label key-value pairs
			labels, err := parseLabelsFromString(labelsString)
			if err != nil {
				t.Errorf("failed to parse labels from %s", filename)
			}

			// replace windows newlines in the json with unix newlines
			expectedCaddyfile = winNewlines.ReplaceAllString(expectedCaddyfile, "\n")

			// convert the labels to a Caddyfile
			caddyfileContainer, err := FromLabels(labels, nil, template.FuncMap{})

			// if the result is nil then we expect an empty Caddyfile
			if caddyfileContainer == nil {
				if expectedCaddyfile != "" {
					t.Errorf("got nil in %s but expected: %s", filename, expectedCaddyfile)
				}
				return
			}

			// if caddyfileContainer is not nil, we expect no error
			assert.NoError(t, err, "expected no error in %s", filename)

			// compare the actual and expected Caddyfiles
			actualCaddyfile := strings.TrimSpace(string(caddyfileContainer.Marshal()))
			assert.Equal(t, expectedCaddyfile, actualCaddyfile,
				"comparison failed in %s: \nExpected:\n%s\n\nActual:\n%s\n",
				filename, expectedCaddyfile, actualCaddyfile)
		})
	}
}

func parseLabelsFromString(s string) (map[string]string, error) {
	labels := make(map[string]string)

	lines := strings.Split(s, "\n")
	lineNumber := 0

	for _, line := range lines {
		line = strings.ReplaceAll(strings.TrimSpace(line), "NEW_LINE", "\n")
		lineNumber++

		// skip lines starting with comment
		if strings.HasPrefix(line, "#") {
			continue
		}

		// skip empty line
		if len(line) == 0 {
			continue
		}

		fields := strings.SplitN(line, "=", 2)
		if len(fields) != 2 {
			return nil, fmt.Errorf("can't parse line %d; line should be in KEY = VALUE format", lineNumber)
		}

		key := strings.TrimSpace(fields[0])
		val := strings.TrimSpace(fields[1])

		if key == "" {
			return nil, fmt.Errorf("missing or empty key on line %d", lineNumber)
		}
		labels[key] = val
	}

	return labels, nil
}
