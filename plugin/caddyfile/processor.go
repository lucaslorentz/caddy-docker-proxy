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

	newCaddyfileBuffer := bytes.Buffer{}
	logsBuffer := bytes.Buffer{}

	container, _ := Unmarshal(caddyfileContent)

	for _, block := range container.Children {
		newContainer := CreateContainer()
		newContainer.AddBlock(block)

		blockCaddyfileContent := newContainer.Marshal()

		adapter := caddyconfig.GetAdapter("caddyfile")

		_, _, err := adapter.Adapt(blockCaddyfileContent, nil)
		if err == nil {
			newCaddyfileBuffer.Write(blockCaddyfileContent)
		} else {
			logsBuffer.WriteString(fmt.Sprintf("[ERROR]  Removing invalid block: %s\n%s\n", err.Error(), blockCaddyfileContent))
		}
	}
	return newCaddyfileBuffer.Bytes(), logsBuffer.Bytes()
}
