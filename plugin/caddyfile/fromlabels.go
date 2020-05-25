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
func FromLabels(labels map[string]string, templateData interface{}, templateFuncs template.FuncMap) (*Block, error) {
	block := CreateBlock()

	for label, value := range labels {
		directive := getOrCreateDirective(block, label)
		argsText, err := processVariables(templateData, templateFuncs, value)
		if err != nil {
			return nil, err
		}
		directive.Args = parseArgs(argsText)
	}

	return block, nil
}

func getOrCreateDirective(directives *Block, path string) *Directive {
	currentBlock := directives
	var directive *Directive
	for _, p := range strings.Split(path, ".") {
		order, name, discriminator := parseLabelSegment(p)
		directive = currentBlock.GetFirstMatch(order, name, discriminator)
		if directive == nil {
			directive = CreateDirective(name, discriminator)
			directive.Order = order
			currentBlock.AddDirective(directive)
		}
		currentBlock = directive.Block
	}
	return directive
}

func parseLabelSegment(text string) (int, string, string) {
	match := labelSegmentRegex.FindStringSubmatch(text)
	order := math.MaxInt32
	if match[1] != "" {
		order, _ = strconv.Atoi(match[1])
	}
	return order, match[2], match[3]
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
