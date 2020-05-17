package caddyfile

import (
	"bytes"
	"math"
	"regexp"
	"sort"
	"strings"
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

// Marshal block into caddyfile bytes
func (block *Block) Marshal() []byte {
	buffer := &bytes.Buffer{}
	block.Write(buffer, true, 0)
	return buffer.Bytes()
}

// MarshalString block into caddyfile string
func (block *Block) MarshalString() string {
	return string(block.Marshal())
}

// Write all directives to a buffer
func (block *Block) Write(buffer *bytes.Buffer, isRoot bool, level int) {
	block.sort()
	for _, subdirective := range block.Children {
		subdirective.Write(buffer, isRoot, level)
	}
}

// Write directive to a buffer
func (directive *Directive) Write(buffer *bytes.Buffer, isRoot bool, identation int) {
	buffer.WriteString(strings.Repeat("\t", identation))
	if isRoot && len(directive.Args) == 0 {
		// This is a global directives block
		directive.Block.Write(buffer, false, identation)
		return
	}
	if !isRoot {
		if directive.Name != "" {
			buffer.WriteString(directive.Name)
		}
		if directive.Name != "" && len(directive.Args) > 0 {
			buffer.WriteString(" ")
		}
	}
	if len(directive.Args) > 0 {
		for index, arg := range directive.Args {
			if index > 0 {
				buffer.WriteString(" ")
			}
			buffer.WriteString(arg)
		}
	}
	if len(directive.Children) > 0 {
		buffer.WriteString(" {\n")
		directive.Block.Write(buffer, false, identation+1)
		buffer.WriteString(strings.Repeat("\t", identation) + "}")
	}
	buffer.WriteString("\n")
}

func (block *Block) sort() {
	items := block.Children
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].isSnippet() && !items[j].isSnippet() {
			return true
		}
		if items[i].Order != items[j].Order {
			return items[i].Order < items[j].Order
		}
		if items[i].Name != items[j].Name {
			return items[i].Name < items[j].Name
		}
		return items[i].Discriminator < items[j].Discriminator
	})
}

func (directive *Directive) isSnippet() bool {
	return len(directive.Args) > 0 && snippetRegex.MatchString(directive.Args[0])
}
