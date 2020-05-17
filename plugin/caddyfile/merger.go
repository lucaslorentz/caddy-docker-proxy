package caddyfile

import "strings"

// Merge a second caddyfile block into the current block
func (blockA *Block) Merge(blockB *Block) {
OuterLoop:
	for _, directiveB := range blockB.Children {
		name := directiveB.Name
		for _, directiveA := range blockA.GetAllByName(name) {
			if name == "reverse_proxy" && getMatcher(directiveA) == getMatcher(directiveB) {
				mergeReverseProxy(directiveA, directiveB)
				continue OuterLoop
			} else if directiveArgsAreEqual(directiveA, directiveB) {
				directiveA.Block.Merge(directiveB.Block)
				continue OuterLoop
			}
		}
		blockA.AddDirective(directiveB)
	}
}

func mergeReverseProxy(directiveA *Directive, directiveB *Directive) {
	for index, arg := range directiveB.Args {
		if index > 0 || !isMatcher(arg) {
			directiveA.AddArgs(arg)
		}
	}
	directiveA.Block.Merge(directiveB.Block)
}

func getMatcher(directive *Directive) string {
	if len(directive.Args) == 0 || !isMatcher(directive.Args[0]) {
		return "*"
	}
	return directive.Args[0]
}

func isMatcher(value string) bool {
	return value == "*" || strings.HasPrefix(value, "/") || strings.HasPrefix(value, "@")
}

func directiveArgsAreEqual(directiveA *Directive, directiveB *Directive) bool {
	if len(directiveA.Args) != len(directiveB.Args) {
		return false
	}
	for i := 0; i < len(directiveA.Args); i++ {
		if directiveA.Args[i] != directiveB.Args[i] {
			return false
		}
	}
	return true
}
