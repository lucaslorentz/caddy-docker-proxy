package caddyfile

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
)

// Process caddyfile and removes wrong server blocks
func Process(caddyfileContent []byte) ([]byte, []byte) {
	logsBuffer := bytes.Buffer{}

	serverBlocks, err := caddyfile.Parse("", caddyfileContent)

	if err != nil {
		logsBuffer.WriteString(fmt.Sprintf("[ERROR] Error parsing caddyfile: %s\n", err.Error()))
	}

	var newCaddyfileBuffer bytes.Buffer

	for _, serverBlock := range serverBlocks {
		serverBlockContent := serializeServerBlock(&serverBlock)

		adapter := caddyconfig.GetAdapter("caddyfile")

		_, _, err := adapter.Adapt(serverBlockContent, nil)
		if err == nil {
			newCaddyfileBuffer.Write(serverBlockContent)
		} else {
			logsBuffer.WriteString(fmt.Sprintf("[ERROR]  Removing invalid server block: %s\n%s\n", err.Error(), serverBlockContent))
		}
	}
	return newCaddyfileBuffer.Bytes(), logsBuffer.Bytes()
}

func serializeServerBlock(serverBlock *caddyfile.ServerBlock) []byte {
	var writer bytes.Buffer
	writeServerBlock(&writer, serverBlock)
	return writer.Bytes()
}

func writeServerBlock(writer *bytes.Buffer, serverBlock *caddyfile.ServerBlock) {
	key := strings.Join(serverBlock.Keys, " ")
	if key != "" {
		writer.WriteString(key)
		writer.WriteString(" ")
	}
	writer.WriteString("{\n")

	for _, segment := range serverBlock.Segments {
		indent := 1
		newLine := true
		tokenLine := -1
		for _, token := range segment {
			if token.Text == "}" {
				indent--
			}
			if token.Line != tokenLine {
				if tokenLine != -1 {
					writer.WriteString("\n")
					newLine = true
				}
				tokenLine = token.Line
			}
			if newLine {
				writer.WriteString(strings.Repeat("\t", indent))
				newLine = false
			} else {
				writer.WriteString(" ")
			}

			if strings.ContainsAny(token.Text, ` "'`) {
				writer.WriteString("\"")
				escapedToken := strings.ReplaceAll(token.Text, "\"", "\\\"")
				writer.WriteString(escapedToken)
				writer.WriteString("\"")
			} else {
				writer.WriteString(token.Text)
			}

			if token.Text == "{" {
				indent++
			}
		}
		writer.WriteString("\n")
	}
	writer.WriteString("}\n")
}
