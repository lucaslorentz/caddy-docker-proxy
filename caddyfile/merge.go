package caddyfile

import "strings"

// Merge a second caddyfile container into this container
func (containerA *Container) Merge(containerB *Container) {
OuterLoop:
	for _, blockB := range containerB.Children {
		firstKey := blockB.GetFirstKey()
		for _, blockA := range containerA.GetAllByFirstKey(firstKey) {
			if (firstKey == "reverse_proxy" || firstKey == "php_fastcgi") && getMatcher(blockA) == getMatcher(blockB) {
				mergeReverseProxyLike(blockA, blockB)
				continue OuterLoop
			} else if blocksAreEqual(blockA, blockB) {
				blockA.Container.Merge(blockB.Container)
				continue OuterLoop
			}
		}
		containerA.AddBlock(blockB)
	}
}

func mergeReverseProxyLike(blockA *Block, blockB *Block) {
	for index, key := range blockB.Keys[1:] {
		if index > 0 || !isMatcher(key) {
			blockA.AddKeys(key)
		}
	}
	blockA.Container.Merge(blockB.Container)
}

func getMatcher(block *Block) string {
	if len(block.Keys) <= 1 || !isMatcher(block.Keys[1]) {
		return "*"
	}
	return block.Keys[1]
}

func isMatcher(value string) bool {
	return value == "*" || strings.HasPrefix(value, "/") || strings.HasPrefix(value, "@")
}

func blocksAreEqual(blockA *Block, blockB *Block) bool {
	if len(blockA.Keys) != len(blockB.Keys) {
		return false
	}
	for i := 0; i < len(blockA.Keys); i++ {
		if blockA.Keys[i] != blockB.Keys[i] {
			return false
		}
	}
	return true
}
