package plugin

import (
	"errors"
	"io/ioutil"
	"os"
	"regexp"
	"runtime"
)

// DockerUtils is an interface with docker utilities
type DockerUtils interface {
	GetCurrentContainerID() (string, error)
}

type dockerUtils struct{}

// CreateDockerUtils creates a new instance of docker utils
func CreateDockerUtils() DockerUtils {
	return &dockerUtils{}
}

// GetCurrentContainerID returns the id of the container running this application
func (wrapper *dockerUtils) GetCurrentContainerID() (string, error) {
	if runtime.GOOS == "windows" {
		return os.Hostname()
	}

	bytes, err := ioutil.ReadFile("/proc/self/cgroup")
	if err != nil {
		return "", err
	}
	if len(bytes) > 0 {
		cgroups := string(bytes)
		idRegex := regexp.MustCompile("docker/([A-Za-z0-9]+)")
		matches := idRegex.FindStringSubmatch(cgroups)
		if len(matches) > 1 {
			return matches[1], nil
		}
	}
	return "", errors.New("Cannot find container id")
}
