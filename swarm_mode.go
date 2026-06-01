package caddydockerproxy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/errdefs"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/docker"

	"go.uber.org/zap"
)

const (
	defaultSwarmCaddyfileTarget = "/etc/caddy/Caddyfile"
	defaultSwarmConfigPrefix    = "caddyfile"
	defaultSwarmConfigHashLen   = 32
	maxSwarmConfigSizeBytes     = 1000 * 1024
)

func (dockerLoader *DockerLoader) updateSwarmService() error {
	ctx := context.Background()
	log := logger()

	svcName := strings.TrimSpace(dockerLoader.options.SwarmService)
	if svcName == "" {
		return fmt.Errorf("swarm service is not set")
	}

	targetPath := strings.TrimSpace(dockerLoader.options.SwarmCaddyfileTarget)
	if targetPath == "" {
		targetPath = defaultSwarmCaddyfileTarget
	}

	prefix := strings.TrimSpace(dockerLoader.options.SwarmConfigPrefix)
	if prefix == "" {
		prefix = defaultSwarmConfigPrefix
	}

	hashLen := dockerLoader.options.SwarmConfigHashLen
	if hashLen == 0 {
		hashLen = defaultSwarmConfigHashLen
	}
	if hashLen < 8 || hashLen > 64 {
		return fmt.Errorf("swarm-config-hash-len must be between 8 and 64 (got %d)", hashLen)
	}

	caddyfile := dockerLoader.lastCaddyfile
	if len(caddyfile) > maxSwarmConfigSizeBytes {
		return fmt.Errorf("generated caddyfile is too large for Swarm config (%d bytes > %d bytes)", len(caddyfile), maxSwarmConfigSizeBytes)
	}

	sum := sha256.Sum256(caddyfile)
	fullHash := hex.EncodeToString(sum[:])
	configName := fmt.Sprintf("%s-%s", prefix, fullHash[:hashLen])

	dockerClient, svc, err := dockerLoader.inspectSwarmService(ctx, svcName)
	if err != nil {
		return err
	}

	configID, created, err := ensureSwarmConfig(ctx, dockerClient, configName, caddyfile, fullHash)
	if err != nil {
		return err
	}
	if created {
		log.Info("Swarm config created", zap.String("name", configName), zap.String("id", configID), zap.String("service", svcName))
	}

	updated, err := dockerLoader.ensureServiceCaddyfileConfig(ctx, dockerClient, svcName, svc, configID, configName, targetPath)
	if err != nil {
		return err
	}
	if updated {
		log.Info("Swarm service updated", zap.String("service", svc.Spec.Name), zap.String("config", configName), zap.String("target", targetPath))
	}

	return nil
}

func (dockerLoader *DockerLoader) inspectSwarmService(ctx context.Context, service string) (docker.Client, swarm.Service, error) {
	log := logger()

	// Try cached client first
	if idx := dockerLoader.swarmServiceClientIndex; idx >= 0 && idx < len(dockerLoader.dockerClients) {
		client := dockerLoader.dockerClients[idx]
		if svc, ok := tryInspectSwarmService(ctx, client, service, log); ok {
			return client, svc, nil
		}

		// Invalidate cache and try all clients
		dockerLoader.swarmServiceClientIndex = -1
	}

	for i, client := range dockerLoader.dockerClients {
		if svc, ok := tryInspectSwarmService(ctx, client, service, log); ok {
			dockerLoader.swarmServiceClientIndex = i
			return client, svc, nil
		}
	}

	return nil, swarm.Service{}, fmt.Errorf("failed to inspect swarm service %q on any configured docker socket", service)
}

func tryInspectSwarmService(ctx context.Context, client docker.Client, service string, log *zap.Logger) (swarm.Service, bool) {
	info, err := client.Info(ctx)
	if err != nil {
		log.Debug("Swarm info check failed", zap.Error(err))
		return swarm.Service{}, false
	}
	if !info.Swarm.ControlAvailable {
		return swarm.Service{}, false
	}

	svc, _, err := client.ServiceInspectWithRaw(ctx, service, swarm.ServiceInspectOptions{})
	if err != nil {
		return swarm.Service{}, false
	}

	return svc, true
}

func ensureSwarmConfig(ctx context.Context, dockerClient docker.Client, name string, data []byte, fullHash string) (string, bool, error) {
	configs, err := dockerClient.ConfigList(ctx, types.ConfigListOptions{})
	if err != nil {
		return "", false, err
	}

	for _, cfg := range configs {
		if cfg.Spec.Name != name {
			continue
		}

		fullCfg, _, err := dockerClient.ConfigInspectWithRaw(ctx, cfg.ID)
		if err != nil {
			return "", false, err
		}

		sum := sha256.Sum256(fullCfg.Spec.Data)
		existingHash := hex.EncodeToString(sum[:])
		if existingHash != fullHash {
			return "", false, fmt.Errorf("swarm config name collision for %q (existing hash %s != desired hash %s); increase --swarm-config-hash-len", name, existingHash, fullHash)
		}

		return cfg.ID, false, nil
	}

	spec := swarm.ConfigSpec{
		Annotations: swarm.Annotations{
			Name:   name,
			Labels: map[string]string{},
		},
		Data: data,
	}

	resp, err := dockerClient.ConfigCreate(ctx, spec)
	if err != nil {
		// If another instance created it concurrently, inspect again.
		if errdefs.IsConflict(err) {
			return ensureSwarmConfig(ctx, dockerClient, name, data, fullHash)
		}
		return "", false, err
	}

	return resp.ID, true, nil
}

func (dockerLoader *DockerLoader) ensureServiceCaddyfileConfig(
	ctx context.Context,
	dockerClient docker.Client,
	svcName string,
	svc swarm.Service,
	configID string,
	configName string,
	targetPath string,
) (bool, error) {
	if strings.TrimSpace(targetPath) == "" {
		return false, fmt.Errorf("swarm caddyfile target is empty")
	}

	const maxAttempts = 5
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Always inspect latest state before updating.
		currentSvc, _, err := dockerClient.ServiceInspectWithRaw(ctx, svcName, swarm.ServiceInspectOptions{})
		if err != nil {
			lastErr = err
			break
		}

		updated, err := updateServiceSpecConfigTarget(&currentSvc.Spec, targetPath, configID, configName)
		if err != nil {
			return false, err
		}
		if !updated {
			return false, nil
		}

		_, err = dockerClient.ServiceUpdate(ctx, currentSvc.ID, currentSvc.Version, currentSvc.Spec, swarm.ServiceUpdateOptions{})
		if err == nil {
			return true, nil
		}

		lastErr = err
		if errdefs.IsConflict(err) {
			time.Sleep(time.Duration(attempt*attempt) * 200 * time.Millisecond)
			continue
		}
		return false, err
	}

	if lastErr != nil {
		return false, lastErr
	}
	return false, fmt.Errorf("failed to update swarm service")
}

func updateServiceSpecConfigTarget(spec *swarm.ServiceSpec, targetPath, configID, configName string) (bool, error) {
	if spec == nil {
		return false, fmt.Errorf("service spec is nil")
	}
	if spec.TaskTemplate.ContainerSpec == nil {
		return false, fmt.Errorf("service has no container spec")
	}

	cs := spec.TaskTemplate.ContainerSpec

	for _, s := range cs.Secrets {
		if s == nil || s.File == nil {
			continue
		}
		if s.File.Name == targetPath {
			return false, fmt.Errorf("service mounts a secret at %q; configs-only swarm mode cannot manage this path", targetPath)
		}
	}

	uid := "0"
	gid := "0"
	mode := os.FileMode(0444)
	seenTarget := 0
	seenDesired := false
	seenOther := false

	for _, cfg := range cs.Configs {
		if cfg == nil || cfg.File == nil {
			continue
		}
		if cfg.File.Name != targetPath {
			continue
		}

		seenTarget++
		if cfg.File.UID != "" {
			uid = cfg.File.UID
		}
		if cfg.File.GID != "" {
			gid = cfg.File.GID
		}
		if cfg.File.Mode != 0 {
			mode = cfg.File.Mode
		}

		if cfg.ConfigID == configID || cfg.ConfigName == configName {
			seenDesired = true
		} else {
			seenOther = true
		}
	}

	if seenTarget == 1 && seenDesired && !seenOther {
		return false, nil
	}

	newConfigs := make([]*swarm.ConfigReference, 0, len(cs.Configs)+1)
	for _, cfg := range cs.Configs {
		if cfg == nil || cfg.File == nil || cfg.File.Name != targetPath {
			newConfigs = append(newConfigs, cfg)
		}
	}

	newConfigs = append(newConfigs, &swarm.ConfigReference{
		ConfigID:   configID,
		ConfigName: configName,
		File: &swarm.ConfigReferenceFileTarget{
			Name: targetPath,
			UID:  uid,
			GID:  gid,
			Mode: mode,
		},
	})

	cs.Configs = newConfigs
	spec.TaskTemplate.ContainerSpec = cs
	return true, nil
}
