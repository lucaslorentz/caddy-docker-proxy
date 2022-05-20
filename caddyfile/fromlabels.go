package caddyfile

import (
	"bytes"
	"math"
	"regexp"
	"strconv"
	"text/template"
)

var whitespaceRegex = regexp.MustCompile("\\s+")
var labelParserRegex = regexp.MustCompile(`^(?:(.+)\.)?(?:(\d+)_)?([^.]+?)(?:_(\d+))?$`)

// FromLabels converts key value labels into a caddyfile
func FromLabels(labels map[string]string, templateData interface{}, templateFuncs template.FuncMap) (*Container, error) {
	container := CreateContainer()

	blocksByPath := map[string]*Block{}
	for label, value := range labels {
		block := getOrCreateBlock(container, label, blocksByPath)
		argsText, err := processVariables(templateData, templateFuncs, value)
		if err != nil {
			return nil, err
		}
		args, err := parseArgs(argsText)
		if err != nil {
			return nil, err
		}
		block.AddKeys(args...)
	}

	return container, nil
}

func getOrCreateBlock(container *Container, path string, blocksByPath map[string]*Block) *Block {
	if block, blockExists := blocksByPath[path]; blockExists {
		return block
	}

	parentPath, order, name := parsePath(path)

	block := CreateBlock()
	block.Order = order

	if parentPath != "" {
		parentBlock := getOrCreateBlock(container, parentPath, blocksByPath)
		block.AddKeys(name)
		parentBlock.AddBlock(block)
	} else {
		container.AddBlock(block)
	}

	blocksByPath[path] = block

	return block
}

func parsePath(path string) (string, int, string) {
	match := labelParserRegex.FindStringSubmatch(path)
	parentPath := match[1]
	order := math.MaxInt32
	if match[2] != "" {
		order, _ = strconv.Atoi(match[2])
	}
	name := match[3]
	return parentPath, order, name
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

func parseArgs(text string) ([]string, error) {
	if len(text) == 0 {
		return []string{}, nil
	}
	l := new(lexer)
	err := l.load(bytes.NewReader([]byte(text)))
	if err != nil {
		return nil, err
	}
	var args []string
	for l.next() {
		args = append(args, l.token.Text)
	}
	return args, nil
}
