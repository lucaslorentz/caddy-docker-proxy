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
	container.sort()
	buffer := &bytes.Buffer{}
	container.write(buffer, 0)
	return buffer.Bytes()
}

// write all blocks to a buffer
func (container *Container) write(buffer *bytes.Buffer, level int) {
	for _, block := range container.Children {
		block.write(buffer, level)
	}
}

// write block to a buffer
func (block *Block) write(buffer *bytes.Buffer, level int) {
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
		block.Container.write(buffer, level+1)
		buffer.WriteString(strings.Repeat("\t", level) + "}")
	}
	buffer.WriteString("\n")
}

func (container *Container) sort() {
	// Sort children first
	for _, block := range container.Children {
		block.Container.sort()
	}
	// Sort container
	items := container.Children
	sort.SliceStable(items, func(i, j int) bool {
		return compareBlocks(items[i], items[j]) == -1
	})
}

func compareBlocks(blockA *Block, blockB *Block) int {
	// Global blocks first
	if blockA.IsGlobalBlock() && !blockB.IsGlobalBlock() {
		return -1
	}
	// Then follow order
	if blockA.Order != blockB.Order {
		if blockA.Order < blockB.Order {
			return -1
		}
		return 1
	}
	// Then compare common keys
	for keyIndex := 0; keyIndex < min(len(blockB.Keys), len(blockB.Keys)); keyIndex++ {
		if blockA.Keys[keyIndex] != blockB.Keys[keyIndex] {
			if blockA.Keys[keyIndex] < blockB.Keys[keyIndex] {
				return -1
			}
			return 1
		}
	}
	// Then the block with less keys first
	if len(blockA.Keys) != len(blockB.Keys) {
		if len(blockA.Keys) < len(blockB.Keys) {
			return -1
		}
		return 1
	}
	// Then based on children
	commonChildrenLength := min(len(blockA.Container.Children), len(blockB.Container.Children))
	for c := 0; c < commonChildrenLength; c++ {
		childComparison := compareBlocks(blockA.Container.Children[c], blockB.Container.Children[c])
		if childComparison != 0 {
			return childComparison
		}
	}
	// Then the block with less children first
	if len(blockA.Container.Children) != len(blockB.Container.Children) {
		if len(blockA.Container.Children) < len(blockB.Container.Children) {
			return -1
		}
		return 1
	}
	return 0
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
