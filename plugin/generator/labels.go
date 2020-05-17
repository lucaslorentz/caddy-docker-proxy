package generator

import (
	"strings"

	"github.com/lucaslorentz/caddy-docker-proxy/v2/plugin/caddyfile"
)

type targetsProvider func() ([]string, error)

func labelsToCaddyfile(labels map[string]string, templateData interface{}, getTargets targetsProvider) (*caddyfile.Block, error) {
	block := caddyfile.FromLabels(labels, templateData)

	// Process special directives
	for _, directive := range block.Children {
		address := directive.GetFirstMatch("address", "")

		if address != nil && len(address.Args) > 0 {
			directive.Args = address.Args

			sourcePath := directive.GetFirstMatch("sourcepath", "")
			targetPort := directive.GetFirstMatch("targetport", "")
			targetPath := directive.GetFirstMatch("targetpath", "")
			targetProtocol := directive.GetFirstMatch("targetprotocol", "")

			proxyDirective := directive.GetOrCreateDirective("reverse_proxy", "")

			if sourcePath != nil && len(sourcePath.Args) > 0 {
				trimmedPath := strings.TrimRight(sourcePath.Args[0], "/")

				routeDirective := directive.GetOrCreateDirective("route", "")
				routeDirective.AddArgs(trimmedPath + "/*")

				stripPrefixDirective := routeDirective.GetOrCreateDirective("uri", "")
				stripPrefixDirective.Order = 1
				stripPrefixDirective.AddArgs("strip_prefix", trimmedPath)

				if targetPath != nil && len(targetPath.Args) > 0 {
					rewriteDirective := routeDirective.GetOrCreateDirective("rewrite", "")
					rewriteDirective.Order = 2
					rewriteDirective.AddArgs("*", strings.TrimRight(targetPath.Args[0], "/")+"{uri}")
				}

				proxyDirective.Order = 3
				directive.Remove(proxyDirective)
				routeDirective.AddDirective(proxyDirective)
			} else if targetPath != nil && len(targetPath.Args) > 0 {
				rewriteDirective := directive.GetOrCreateDirective("rewrite", "")
				rewriteDirective.AddArgs("*", strings.TrimRight(targetPath.Args[0], "/")+"{uri}")
			}

			if len(proxyDirective.Args) == 0 {
				proxyTargets, err := getTargets()
				if err != nil {
					return nil, err
				}

				for _, target := range proxyTargets {
					targetArg := ""
					if targetProtocol != nil && len(targetProtocol.Args) > 0 {
						targetArg += targetProtocol.Args[0] + "://"
					}

					targetArg += target

					if targetPort != nil && len(targetPort.Args) > 0 {
						targetArg += ":" + targetPort.Args[0]
					}

					proxyDirective.AddArgs(targetArg)
				}
			}
		}

		directive.RemoveAllMatches("address", "")
		directive.RemoveAllMatches("sourcepath", "")
		directive.RemoveAllMatches("targetport", "")
		directive.RemoveAllMatches("targetpath", "")
		directive.RemoveAllMatches("targetprotocol", "")
	}

	return block, nil
}
