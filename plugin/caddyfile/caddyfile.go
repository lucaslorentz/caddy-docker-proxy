package caddyfile

import (
	"math"
	"regexp"
)

var snippetRegex = regexp.MustCompile(`^\(.*\)$`)

// Block represents any element in a caddyfile:
// - GlobalOptions
// - Snippet
// - Site
// - MatcherDefinition
// - Option
// - Directive
// - Subdirective
// It's structure is rendering in a caddyfile as:
// Name Args[0] Args[1] ... {
//	...Children
// }
type Block struct {
	*Container
	Order         int
	Name          string
	Discriminator string
	Args          []string
}

// Container represents a collection of blocks
type Container struct {
	Children []*Block
}

// CreateBlock creates a block with a name and a discriminator
func CreateBlock(name string, discriminator string) *Block {
	return &Block{
		Container:     CreateContainer(),
		Order:         math.MaxInt32,
		Name:          name,
		Discriminator: discriminator,
	}
}

// CreateContainer creates a block container
func CreateContainer() *Container {
	return &Container{
		Children: []*Block{},
	}
}

// AddArgs add one or more arguments to block
func (block *Block) AddArgs(args ...string) {
	block.Args = append(block.Args, args...)
}

// AddBlock adds a block to a container
func (container *Container) AddBlock(block *Block) {
	container.Children = append(container.Children, block)
}

// GetOrCreateBlock gets an existing block or create a new one if not found
func (container *Container) GetOrCreateBlock(order int, name string, discriminator string) *Block {
	existing := container.GetFirstMatch(order, name, discriminator)
	if existing == nil {
		existing = CreateBlock(name, discriminator)
		container.AddBlock(existing)
	}
	return existing
}

// GetFirstMatch gets the first block that matches parameters
func (container *Container) GetFirstMatch(order int, name string, discriminator string) *Block {
	for _, block := range container.Children {
		if block.Order == order && block.Name == name && block.Discriminator == discriminator {
			return block
		}
	}
	return nil
}

// GetAllByName gets all blocks with that name
func (container *Container) GetAllByName(name string) []*Block {
	matched := []*Block{}
	for _, block := range container.Children {
		if block.Name == name {
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

// RemoveAllMatches removes all matching blocks
func (container *Container) RemoveAllMatches(name string, discriminator string) {
	newItems := []*Block{}
	for _, block := range container.Children {
		if block.Name != name || block.Discriminator != discriminator {
			newItems = append(newItems, block)
		}
	}
	container.Children = newItems
}

// IsGlobalBlock returns if block is a global block
func (block *Block) IsGlobalBlock() bool {
	return len(block.Args) == 0
}
