package caddyfile

import (
	"bytes"
	"sort"
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
	container.sort()
	for _, block := range container.Children {
		block.Write(buffer, level)
	}
}

// Write block to a buffer
func (block *Block) Write(buffer *bytes.Buffer, level int) {
	buffer.WriteString(strings.Repeat("\t", level))
	needsWhitespace := false
	for _, key := range block.Keys {
		if needsWhitespace {
			buffer.WriteString(" ")
		}

		if strings.ContainsAny(key, ` "'`) {
			buffer.WriteString("\"")
			buffer.WriteString(strings.ReplaceAll(key, "\"", "\\\""))
			buffer.WriteString("\"")
		} else {
			buffer.WriteString(key)
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

func (container *Container) sort() {
	items := container.Children
	sort.SliceStable(items, func(i, j int) bool {
		// Global blocks first
		if items[i].IsGlobalBlock() && !items[j].IsGlobalBlock() {
			return true
		}
		// Then follow order
		if items[i].Order != items[j].Order {
			return items[i].Order < items[j].Order
		}
		// Then compare common keys
		for keyIndex := 0; keyIndex < min(len(items[i].Keys), len(items[j].Keys)); keyIndex++ {
			if items[i].Keys[keyIndex] != items[j].Keys[keyIndex] {
				return items[i].Keys[keyIndex] < items[j].Keys[keyIndex]
			}
		}
		// Then the block with less keys first
		return len(items[i].Keys) < len(items[j].Keys)
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Unmarshal a Block fom caddyfile content
func Unmarshal(caddyfileContent []byte) (*Container, error) {
	serverBlocks, err := caddyfile.Parse("", caddyfileContent)
	if err != nil {
		return nil, err
	}

	container := CreateContainer()
	for index, serverBlock := range serverBlocks {
		container.AddBlock(unmarshalBlock(&serverBlock, index))
	}
	return container, nil
}

func unmarshalBlock(serverBlock *caddyfile.ServerBlock, index int) *Block {
	stack := []*Block{}

	block := CreateBlock()
	block.Order = index
	block.AddKeys(serverBlock.Keys...)
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
				subBlock = CreateBlock()
				subBlock.Order = len(parentBlock.Children)
				subBlock.AddKeys(token.Text)
				parentBlock.AddBlock(subBlock)
				isNewBlock = false
			} else {
				subBlock.AddKeys(token.Text)
			}
		}
	}

	return block
}
