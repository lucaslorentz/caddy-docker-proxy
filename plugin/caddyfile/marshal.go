package caddyfile

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	_ "github.com/caddyserver/caddy/v2/modules/standard" // plug standard HTTP modules
)

// Marshal container into caddyfile bytes
func (container *Container) Marshal() []byte {
	container.sort()
	buffer := &bytes.Buffer{}
	container.write(buffer, 0)
	return buffer.Bytes()
}

// Marshal block into caddyfile bytes
func (block *Block) Marshal() []byte {
	block.Container.sort()
	buffer := &bytes.Buffer{}
	block.write(buffer, 0)
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

		if strings.ContainsAny(key, "\n\"") {
			// If token has line break or quote, we use backtick for readability
			buffer.WriteString("`")
			buffer.WriteString(strings.ReplaceAll(key, "`", "\\`"))
			buffer.WriteString("`")
		} else if strings.ContainsAny(key, ` `) {
			// If token has whitespace, we use duoble quote
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
	if blockA.IsGlobalBlock() != blockB.IsGlobalBlock() {
		if blockA.IsGlobalBlock() {
			return -1
		}
		return 1
	}
	// Then snippets first
	if blockA.IsSnippet() != blockB.IsSnippet() {
		if blockA.IsSnippet() {
			return -1
		}
		return 1
	}
	// Then follow order
	if blockA.Order != blockB.Order {
		if blockA.Order < blockB.Order {
			return -1
		}
		return 1
	}
	// Then compare common keys
	for keyIndex := 0; keyIndex < min(len(blockA.Keys), len(blockB.Keys)); keyIndex++ {
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
	tokens, err := allTokens("", caddyfileContent)
	if err != nil {
		return nil, err
	}

	return parseContainer(tokens)
}

func allTokens(filename string, input []byte) ([]Token, error) {
	l := new(lexer)
	err := l.load(bytes.NewReader(input))
	if err != nil {
		return nil, err
	}
	var tokens []Token
	for l.next() {
		l.token.File = filename
		tokens = append(tokens, l.token)
	}
	return tokens, nil
}

func parseContainer(tokens []Token) (*Container, error) {
	rootContainer := CreateContainer()
	stack := []*Container{rootContainer}
	isNewBlock := true
	tokenLine := -1

	var currentBlock *Block

	for _, token := range tokens {
		if token.Line != tokenLine {
			if tokenLine != -1 {
				isNewBlock = true
			}
			tokenLine = token.Line
		}
		if token.Text == "}" {
			if len(stack) == 1 {
				return nil, fmt.Errorf("Unexpected token '}' at line %v", token.Line)
			}
			stack = stack[:len(stack)-1]
		} else {
			if isNewBlock {
				parentBlock := stack[len(stack)-1]
				currentBlock = CreateBlock()
				currentBlock.Order = len(parentBlock.Children)
				parentBlock.AddBlock(currentBlock)
				isNewBlock = false
			}
			if token.Text == "{" {
				stack = append(stack, currentBlock.Container)
			} else {
				currentBlock.AddKeys(token.Text)
				tokenLine += strings.Count(token.Text, "\n")
			}
		}
	}

	return rootContainer, nil
}
