package plugin

import (
	"bytes"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/caddyserver/caddy"
	"github.com/caddyserver/caddy/caddyfile"

	_ "github.com/caddyserver/caddy/caddyhttp" // plug in the HTTP server type
)

// ProcessCaddyfile validate and removes wrong server blocks from caddyfile
func ProcessCaddyfile(caddyfileContent []byte) []byte {
	serverBlocks, err := caddyfile.Parse("", bytes.NewReader(caddyfileContent), nil)

	if err != nil {
		log.Printf("[ERROR] Error parsing caddyfile:%s\n", err)
	}

	var validServerBlocks []caddyfile.ServerBlock
	for _, serverBlock := range serverBlocks {
		serverBlockContent := serializeServerBlock(&serverBlock)

		newInput := caddy.CaddyfileInput{
			ServerTypeName: "http",
			Contents:       serverBlockContent,
		}

		err := caddy.ValidateAndExecuteDirectives(newInput, nil, true)
		if err == nil {
			validServerBlocks = append(validServerBlocks, serverBlock)
		} else {
			log.Printf("[WARN] Removing invalid server block: %s\n%s\n", err, serverBlockContent)
		}
	}
	return serializeServerBlocks(validServerBlocks)
}

func serializeServerBlocks(serverBlocks []caddyfile.ServerBlock) []byte {
	var writer bytes.Buffer

	for _, serverBlock := range serverBlocks {
		writeServerBlock(&writer, &serverBlock)
	}

	return writer.Bytes()
}

func serializeServerBlock(serverBlock *caddyfile.ServerBlock) []byte {
	var writer bytes.Buffer
	writeServerBlock(&writer, serverBlock)
	return writer.Bytes()
}

func writeServerBlock(writer *bytes.Buffer, serverBlock *caddyfile.ServerBlock) {
	writer.WriteString(fmt.Sprintf("%v {\n", strings.Join(serverBlock.Keys, " ")))

	for _, directiveName := range getSortedDirectiveNames(serverBlock.Tokens) {
		indent := 1
		newLine := true
		tokenLine := -1
		for _, token := range serverBlock.Tokens[directiveName] {
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
				writer.WriteString(strings.Repeat("  ", indent))
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

func getSortedDirectiveNames(m map[string][]caddyfile.Token) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
