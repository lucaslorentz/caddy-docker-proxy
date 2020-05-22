package caddyfile

import (
	"bytes"
	"sort"
	"strings"

	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	_ "github.com/caddyserver/caddy/v2/modules/standard" // plug standard HTTP modules
)

// Marshal block into caddyfile bytes
func (block *Block) Marshal() []byte {
	buffer := &bytes.Buffer{}
	block.Write(buffer, 0)
	return buffer.Bytes()
}

// MarshalString block into caddyfile string
func (block *Block) MarshalString() string {
	return string(block.Marshal())
}

// Write all directives to a buffer
func (block *Block) Write(buffer *bytes.Buffer, level int) {
	block.sort(level)
	for _, subdirective := range block.Children {
		subdirective.Write(buffer, level)
	}
}

// Write directive to a buffer
func (directive *Directive) Write(buffer *bytes.Buffer, level int) {
	buffer.WriteString(strings.Repeat("\t", level))
	needsWhitespace := false
	if level > 0 && directive.Name != "" {
		buffer.WriteString(directive.Name)
		needsWhitespace = true
	}
	for _, arg := range directive.Args {
		if needsWhitespace {
			buffer.WriteString(" ")
		}

		if strings.ContainsAny(arg, ` "'`) {
			buffer.WriteString("\"")
			buffer.WriteString(strings.ReplaceAll(arg, "\"", "\\\""))
			buffer.WriteString("\"")
		} else {
			buffer.WriteString(arg)
		}

		needsWhitespace = true
	}
	if len(directive.Children) > 0 {
		if needsWhitespace {
			buffer.WriteString(" ")
		}
		buffer.WriteString("{\n")
		directive.Block.Write(buffer, level+1)
		buffer.WriteString(strings.Repeat("\t", level) + "}")
	}
	buffer.WriteString("\n")
}

func (block *Block) sort(level int) {
	items := block.Children
	sort.SliceStable(items, func(i, j int) bool {
		if level == 0 && items[i].IsGlobalBlock() && !items[j].IsGlobalBlock() {
			return true
		}
		if items[i].Order != items[j].Order {
			return items[i].Order < items[j].Order
		}
		if items[i].Name != items[j].Name {
			return items[i].Name < items[j].Name
		}
		if len(items[i].Args) > 0 && len(items[j].Args) > 0 && items[i].Args[0] != items[j].Args[0] {
			return items[i].Args[0] < items[j].Args[0]
		}
		return items[i].Discriminator < items[j].Discriminator
	})
}

// Unmarshal a Block fom caddyfile content
func Unmarshal(caddyfileContent []byte) (*Block, error) {
	serverBlocks, err := caddyfile.Parse("", caddyfileContent)
	if err != nil {
		return nil, err
	}

	block := CreateBlock()
	for index, serverBlock := range serverBlocks {
		block.AddDirective(unmarshalServerBlock(&serverBlock, index))
	}
	return block, nil
}

func unmarshalServerBlock(serverBlock *caddyfile.ServerBlock, index int) *Directive {
	stack := []*Directive{}

	directive := CreateDirective("caddy", "")
	directive.Order = index
	directive.AddArgs(serverBlock.Keys...)
	stack = append(stack, directive)

	for _, segment := range serverBlock.Segments {
		newDirective := true
		tokenLine := -1
		var subDirective *Directive

		for _, token := range segment {
			if token.Line != tokenLine {
				if tokenLine != -1 {
					newDirective = true
				}
				tokenLine = token.Line
			}
			if token.Text == "}" {
				stack = stack[:len(stack)-1]
			} else if token.Text == "{" {
				stack = append(stack, subDirective)
			} else if newDirective {
				subDirective = CreateDirective(token.Text, "")
				subDirective.Order = len(stack[len(stack)-1].Children)
				stack[len(stack)-1].AddDirective(subDirective)
				newDirective = false
			} else {
				subDirective.AddArgs(token.Text)
			}
		}
	}

	return directive
}
