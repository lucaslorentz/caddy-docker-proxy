package caddyfile

import (
	"bytes"
	"sort"
	"strconv"
	"strings"

	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	_ "github.com/caddyserver/caddy/v2/modules/standard" // plug standard HTTP modules
)

// Marshal container into caddyfile bytes
func (container *Container) Marshal() []byte {
	buffer := &bytes.Buffer{}
	container.Write(buffer, 0)
	return buffer.Bytes()
}

// Write all blocks to a buffer
func (container *Container) Write(buffer *bytes.Buffer, level int) {
	container.sort(level)
	for _, block := range container.Children {
		block.Write(buffer, level)
	}
}

// Write block to a buffer
func (block *Block) Write(buffer *bytes.Buffer, level int) {
	buffer.WriteString(strings.Repeat("\t", level))
	needsWhitespace := false
	if level > 0 && block.Name != "" {
		buffer.WriteString(block.Name)
		needsWhitespace = true
	}
	for _, arg := range block.Args {
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
	if len(block.Children) > 0 {
		if needsWhitespace {
			buffer.WriteString(" ")
		}
		buffer.WriteString("{\n")
		block.Container.Write(buffer, level+1)
		buffer.WriteString(strings.Repeat("\t", level) + "}")
	}
	buffer.WriteString("\n")
}

func (container *Container) sort(level int) {
	items := container.Children
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
func Unmarshal(caddyfileContent []byte) (*Container, error) {
	serverBlocks, err := caddyfile.Parse("", caddyfileContent)
	if err != nil {
		return nil, err
	}

	container := CreateContainer()
	for index, serverBlock := range serverBlocks {
		container.AddBlock(unmarshalServerBlock(&serverBlock, index))
	}
	return container, nil
}

func unmarshalServerBlock(serverBlock *caddyfile.ServerBlock, index int) *Block {
	stack := []*Block{}

	block := CreateBlock("", strconv.Itoa(index))
	block.Order = index
	block.AddArgs(serverBlock.Keys...)
	stack = append(stack, block)

	for _, segment := range serverBlock.Segments {
		isNewBlock := true
		tokenLine := -1
		var subBlock *Block

		for _, token := range segment {
			if token.Line != tokenLine {
				if tokenLine != -1 {
					isNewBlock = true
				}
				tokenLine = token.Line
			}
			if token.Text == "}" {
				stack = stack[:len(stack)-1]
			} else if token.Text == "{" {
				stack = append(stack, subBlock)
			} else if isNewBlock {
				parentBlock := stack[len(stack)-1]
				subBlock = CreateBlock(token.Text, "")
				subBlock.Order = len(parentBlock.Children)
				parentBlock.AddBlock(subBlock)
				isNewBlock = false
			} else {
				subBlock.AddArgs(token.Text)
			}
		}
	}

	return block
}
