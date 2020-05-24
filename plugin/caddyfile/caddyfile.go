package caddyfile

import (
	"math"
	"regexp"
)

var snippetRegex = regexp.MustCompile(`^\(.*\)$`)

// Directive represents a segment in caddyfile
type Directive struct {
	*Block
	Order         int
	Name          string
	Discriminator string
	Args          []string
}

// Block represents a collection of directives
type Block struct {
	Children []*Directive
}

// CreateDirective creates a directive with a name and a discriminator
func CreateDirective(name string, discriminator string) *Directive {
	return &Directive{
		Block:         CreateBlock(),
		Order:         math.MaxInt32,
		Name:          name,
		Discriminator: discriminator,
	}
}

// CreateBlock creates a directive container
func CreateBlock() *Block {
	return &Block{
		Children: []*Directive{},
	}
}

// AddArgs add one or more arguments to directive
func (directive *Directive) AddArgs(args ...string) {
	directive.Args = append(directive.Args, args...)
}

// AddDirective adds a directive to a container
func (block *Block) AddDirective(directive *Directive) {
	block.Children = append(block.Children, directive)
}

// GetOrCreateDirective gets an existing directive or create a new one if not found
func (block *Block) GetOrCreateDirective(name string, discriminator string) *Directive {
	existing := block.GetFirstMatch(name, discriminator)
	if existing == nil {
		existing = CreateDirective(name, discriminator)
		block.AddDirective(existing)
	}
	return existing
}

// GetFirstMatch gets the first subdirective that matches parameters
func (block *Block) GetFirstMatch(name string, discriminator string) *Directive {
	for _, directive := range block.Children {
		if directive.Name == name && directive.Discriminator == discriminator {
			return directive
		}
	}
	return nil
}

// GetAllByName gets all subdirectives with that name
func (block *Block) GetAllByName(name string) []*Directive {
	matched := []*Directive{}
	for _, directive := range block.Children {
		if directive.Name == name {
			matched = append(matched, directive)
		}
	}
	return matched
}

// Remove removes a specific subdirective
func (block *Block) Remove(directiveToDelete *Directive) {
	newItems := []*Directive{}
	for _, directive := range block.Children {
		if directive != directiveToDelete {
			newItems = append(newItems, directive)
		}
	}
	block.Children = newItems
}

// RemoveAllMatches removes all matching subdirectives
func (block *Block) RemoveAllMatches(name string, discriminator string) {
	newItems := []*Directive{}
	for _, directive := range block.Children {
		if directive.Name != name || directive.Discriminator != discriminator {
			newItems = append(newItems, directive)
		}
	}
	block.Children = newItems
}

// IsGlobalBlock returns if directive is global directive
func (directive *Directive) IsGlobalBlock() bool {
	return len(directive.Args) == 0
}
