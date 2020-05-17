package caddyfile

import (
	"bytes"
	"log"
	"math"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

var whitespaceRegex = regexp.MustCompile("\\s+")
var labelSegmentRegex = regexp.MustCompile(`^(?:(\d+)_)?(.*?)(?:_(\d+))?$`)

// FromLabels converts key value labels into a caddyfile
func FromLabels(labels map[string]string, templateData interface{}) *Block {
	block := CreateBlock()

	for label, value := range labels {
		directive := getOrCreateDirective(block, label)
		argsText := processVariables(templateData, value)
		directive.Args = parseArgs(argsText)
	}

	return block
}

func getOrCreateDirective(directives *Block, path string) *Directive {
	currentBlock := directives
	var directive *Directive
	for _, p := range strings.Split(path, ".") {
		order, name, discriminator := parseLabelSegment(p)
		directive = currentBlock.GetFirstMatch(name, discriminator)
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

func processVariables(data interface{}, content string) string {
	t, err := template.New("").Parse(content)
	if err != nil {
		log.Println(err)
		return content
	}
	var writer bytes.Buffer
	t.Execute(&writer, data)
	return writer.String()
}

func parseArgs(text string) []string {
	args := whitespaceRegex.Split(text, -1)
	if len(args) == 1 && args[0] == "" {
		return []string{}
	}
	return args
}
