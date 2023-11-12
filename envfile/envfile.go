// envfile provides a capability of maing environment variables
// from a file accessible within the application.

package envfile

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"os"
)

func LoadFrom(envFile string) error {
	file, err := os.Open(envFile)
	if err != nil {
		return fmt.Errorf("reading environment file: %v", err)
	}
	defer file.Close()

	envMap, err := parseEnvFile(file)
	if err != nil {
		return fmt.Errorf("parsing environment file: %v", err)
	}

	for k, v := range envMap {
		// do not overwrite existing environment variables
		_, exists := os.LookupEnv(k)
		if !exists {
			if err := os.Setenv(k, v); err != nil {
				return fmt.Errorf("setting environment variables: %v", err)
			}
		}
	}

	return nil
}

// parseEnvFile parses an env file from KEY=VALUE format.
// It's pretty naive. Limited value quotation is supported,
// but variable and command expansions are not supported.
func parseEnvFile(envInput io.Reader) (map[string]string, error) {
	envMap := make(map[string]string)

	scanner := bufio.NewScanner(envInput)
	var lineNumber int

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lineNumber++

		// skip empty lines and lines starting with comment
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// split line into key and value
		before, after, isCut := strings.Cut(line, "=")
		if !isCut {
			return nil, fmt.Errorf("can't parse line %d; line should be in KEY=VALUE format", lineNumber)
		}
		key, val := before, after

		// sometimes keys are prefixed by "export " so file can be sourced in bash; ignore it here
		key = strings.TrimPrefix(key, "export ")

		// validate key and value
		if key == "" {
			return nil, fmt.Errorf("missing or empty key on line %d", lineNumber)
		}
		if strings.Contains(key, " ") {
			return nil, fmt.Errorf("invalid key on line %d: contains whitespace: %s", lineNumber, key)
		}
		if strings.HasPrefix(val, " ") || strings.HasPrefix(val, "\t") {
			return nil, fmt.Errorf("invalid value on line %d: whitespace before value: '%s'", lineNumber, val)
		}

		// remove any trailing comment after value
		if commentStart, _, found := strings.Cut(val, "#"); found {
			val = strings.TrimRight(commentStart, " \t")
		}

		// quoted value: support newlines
		if strings.HasPrefix(val, `"`) || strings.HasPrefix(val, "'") {
			quote := string(val[0])
			for !(strings.HasSuffix(line, quote) && !strings.HasSuffix(line, `\`+quote)) {
				val = strings.ReplaceAll(val, `\`+quote, quote)
				if !scanner.Scan() {
					break
				}
				lineNumber++
				line = strings.ReplaceAll(scanner.Text(), `\`+quote, quote)
				val += "\n" + line
			}
			val = strings.TrimPrefix(val, quote)
			val = strings.TrimSuffix(val, quote)
		}

		envMap[key] = val
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return envMap, nil
}
