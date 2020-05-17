package caddyfile

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
)

// Process caddyfile and removes wrong server blocks
func Process(caddyfileContent []byte) []byte {
	serverBlocks, err := caddyfile.Parse("", caddyfileContent)

	if err != nil {
		log.Printf("[ERROR] Error parsing caddyfile: %s\n", err)
	}

	var newCaddyfileBuffer bytes.Buffer

	for _, serverBlock := range serverBlocks {
		serverBlockContent := serializeServerBlock(&serverBlock)

		adapter := caddyconfig.GetAdapter("caddyfile")

		_, _, err := adapter.Adapt(serverBlockContent, nil)
		if err == nil {
			newCaddyfileBuffer.Write(serverBlockContent)
		} else {
			log.Printf("[WARN] Removing invalid server block: %s\n%s\n", err, serverBlockContent)
		}
	}
	return newCaddyfileBuffer.Bytes()
}

func serializeServerBlock(serverBlock *caddyfile.ServerBlock) []byte {
	var writer bytes.Buffer
	writeServerBlock(&writer, serverBlock)
	return writer.Bytes()
}

func writeServerBlock(writer *bytes.Buffer, serverBlock *caddyfile.ServerBlock) {
	writer.WriteString(fmt.Sprintf("%v {\n", strings.Join(serverBlock.Keys, " ")))

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

			escapedTokenText := strings.ReplaceAll(token.Text, "\"", "\\\"")
			if strings.ContainsAny(token.Text, " ") {
				writer.WriteString("\"")
				writer.WriteString(escapedTokenText)
				writer.WriteString("\"")
			} else {
				writer.WriteString(escapedTokenText)
			}

			if token.Text == "{" {
				indent++
			}
		}
		writer.WriteString("\n")
	}
	writer.WriteString("}\n")
}
