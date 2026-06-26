package caddydockerproxy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/caddyserver/caddy/v2"
)

// pushLocal loads the config into the in-process Caddy via caddy.Load, the same
// function the admin /load handler calls. The loader runs in the same process
// as the local Caddy, so this avoids looping back through the admin API over
// HTTP - dropping the dependency on the local admin endpoint being reachable, or
// even enabled. forceReload is false so an unchanged config is a no-op, matching
// the admin /load default. The recover keeps parity with the HTTP path: the
// admin endpoint's net/http handler recovers panics during provisioning, so a
// poison config must fail the reload here instead of crashing the process.
func pushLocal(postBody []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic loading config locally: %v", r)
		}
	}()
	return caddy.Load(postBody, false)
}

// pushRemoteAdmin POSTs the config to a controlled server's admin API.
func pushRemoteAdmin(server string, postBody []byte) error {
	url := "http://" + server + ":2019/load"

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(postBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, bodyBytes)
	}

	return nil
}
