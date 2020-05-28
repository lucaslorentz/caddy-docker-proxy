package caddyfile

import "strings"

// Merge a second caddyfile container into the current container
func (containerA *Container) Merge(containerB *Container) {
OuterLoop:
	for _, blockB := range containerB.Children {
		name := blockB.Name
		for _, blockA := range containerA.GetAllByName(name) {
			if name == "reverse_proxy" && getMatcher(blockA) == getMatcher(blockB) {
				mergeReverseProxy(blockA, blockB)
				continue OuterLoop
			} else if blocksAreEqual(blockA, blockB) {
				blockA.Container.Merge(blockB.Container)
				continue OuterLoop
			}
		}
		containerA.AddBlock(blockB)
	}
}

func mergeReverseProxy(blockA *Block, blockB *Block) {
	for index, arg := range blockB.Args {
		if index > 0 || !isMatcher(arg) {
			blockA.AddArgs(arg)
		}
	}
	blockA.Container.Merge(blockB.Container)
}

func getMatcher(block *Block) string {
	if len(block.Args) == 0 || !isMatcher(block.Args[0]) {
		return "*"
	}
	return block.Args[0]
}

func isMatcher(value string) bool {
	return value == "*" || strings.HasPrefix(value, "/") || strings.HasPrefix(value, "@")
}

func blocksAreEqual(blockA *Block, blockB *Block) bool {
	if len(blockA.Args) != len(blockB.Args) {
		return false
	}
	for i := 0; i < len(blockA.Args); i++ {
		if blockA.Args[i] != blockB.Args[i] {
			return false
		}
	}
	return true
}
