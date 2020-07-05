package caddyfile

import (
	"math"
	"strings"
)

// Block can represent any of those caddyfile elements:
// - GlobalOptions
// - Snippet
// - Site
// - MatcherDefinition
// - Option
// - Directive
// - Subdirective
//
// It's structure is rendered in caddyfile as:
// Keys[0] Keys[1] Keys[2]
//
// When children are defined, each child is also recursively rendered as:
// Keys[0] Keys[1] Keys[2] {
//	Children[0].Keys[0] Children[0].Keys[1]
//	Children[1].Keys[0] Children[1].Keys[1]
// }
type Block struct {
	*Container
	Order int
	Keys  []string
}

// Container represents a collection of blocks
type Container struct {
	Children []*Block
}

// CreateBlock creates a block
func CreateBlock() *Block {
	return &Block{
		Container: CreateContainer(),
		Order:     math.MaxInt32,
		Keys:      []string{},
	}
}

// CreateContainer creates a container
func CreateContainer() *Container {
	return &Container{
		Children: []*Block{},
	}
}

// AddKeys to block
func (block *Block) AddKeys(keys ...string) {
	block.Keys = append(block.Keys, keys...)
}

// AddBlock to container
func (container *Container) AddBlock(block *Block) {
	container.Children = append(container.Children, block)
}

// GetFirstKey from block
func (block *Block) GetFirstKey() string {
	if len(block.Keys) == 0 {
		return ""
	}
	return block.Keys[0]
}

// GetAllByFirstKey gets all blocks with the specified firstKey
func (container *Container) GetAllByFirstKey(firstKey string) []*Block {
	matched := []*Block{}
	for _, block := range container.Children {
		if block.GetFirstKey() == firstKey {
			matched = append(matched, block)
		}
	}
	return matched
}

// Remove removes a specific block
func (container *Container) Remove(blockToDelete *Block) {
	newItems := []*Block{}
	for _, block := range container.Children {
		if block != blockToDelete {
			newItems = append(newItems, block)
		}
	}
	container.Children = newItems
}

// IsGlobalBlock returns if block is a global block
func (block *Block) IsGlobalBlock() bool {
	return len(block.Keys) == 0
}

// IsSnippet returns if block is a snippet
func (block *Block) IsSnippet() bool {
	return len(block.Keys) == 1 && strings.HasPrefix(block.Keys[0], "(") && strings.HasSuffix(block.Keys[0], ")")
}
