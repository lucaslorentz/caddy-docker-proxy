package caddyfile

import (
	"bytes"
	"fmt"

	"github.com/caddyserver/caddy/v2/caddyconfig"
)

// Process caddyfile and removes wrong server blocks
func Process(caddyfileContent []byte) ([]byte, []byte) {
	if len(caddyfileContent) == 0 {
		return caddyfileContent, nil
	}

	logsBuffer := bytes.Buffer{}
	adapter := caddyconfig.GetAdapter("caddyfile")

	container, err := Unmarshal(caddyfileContent)
	if err != nil {
		logsBuffer.WriteString(fmt.Sprintf("[ERROR]  Invalid caddyfile: %s\n%s\n", err.Error(), caddyfileContent))
		return nil, logsBuffer.Bytes()
	}

	newContainer := CreateContainer()

	container.sort()
	for _, block := range container.Children {
		newContainer.AddBlock(block)

		_, _, err := adapter.Adapt(newContainer.Marshal(), nil)

		if err != nil {
			newContainer.Remove(block)
			logsBuffer.WriteString(fmt.Sprintf("[ERROR]  Removing invalid block: %s\n%s\n", err.Error(), block.Marshal()))
		}
	}

	return newContainer.Marshal(), logsBuffer.Bytes()
}
