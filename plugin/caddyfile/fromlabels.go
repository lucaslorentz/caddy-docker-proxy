package caddyfile

import (
	"bytes"
	"math"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

var whitespaceRegex = regexp.MustCompile("\\s+")
var labelSegmentRegex = regexp.MustCompile(`^(?:(\d+)_)?(.*?)(?:_(\d+))?$`)

// FromLabels converts key value labels into a caddyfile
func FromLabels(labels map[string]string, templateData interface{}, templateFuncs template.FuncMap) (*Container, error) {
	container := CreateContainer()

	for label, value := range labels {
		block := getOrCreateBlock(container, label)
		argsText, err := processVariables(templateData, templateFuncs, value)
		if err != nil {
			return nil, err
		}
		block.Args = parseArgs(argsText)
	}

	return container, nil
}

func getOrCreateBlock(container *Container, path string) *Block {
	currentContainer := container
	var block *Block
	for i, p := range strings.Split(path, ".") {
		order, name, discriminator := parseLabelSegment(p, i)
		block = currentContainer.GetFirstMatch(order, name, discriminator)
		if block == nil {
			block = CreateBlock(name, discriminator)
			block.Order = order
			currentContainer.AddBlock(block)
		}
		currentContainer = block.Container
	}
	return block
}

func parseLabelSegment(text string, index int) (int, string, string) {
	match := labelSegmentRegex.FindStringSubmatch(text)
	order := math.MaxInt32
	name := ""
	if match[1] != "" {
		order, _ = strconv.Atoi(match[1])
	}
	if index > 0 {
		name = match[2]
	}
	return order, name, match[3]
}

func processVariables(data interface{}, funcs template.FuncMap, content string) (string, error) {
	t, err := template.New("").Funcs(funcs).Parse(content)
	if err != nil {
		return "", err
	}
	var writer bytes.Buffer
	err = t.Execute(&writer, data)
	if err != nil {
		return "", err
	}
	return writer.String(), nil
}

func parseArgs(text string) []string {
	args := whitespaceRegex.Split(text, -1)
	if len(args) == 1 && args[0] == "" {
		return []string{}
	}
	return args
}
