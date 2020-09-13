package docker

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"runtime"
)

// Utils is an interface with docker utilities
type Utils interface {
	GetCurrentContainerID() (string, error)
}

type dockerUtils struct{}

// CreateUtils creates a new instance of docker utils
func CreateUtils() Utils {
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
	if len(bytes) == 0 {
		return "", errors.New("Cannot read /proc/self/cgroup")
	}

	return wrapper.ExtractContainerID(string(bytes))

}

func (wrapper *dockerUtils) ExtractContainerID(cgroups string) (string, error) {
	idRegex := regexp.MustCompile(`(?i):[^:]*\bcpu\b[^:]*:[^/]*/.*([[:alnum:]]{64}).*`)
	matches := idRegex.FindStringSubmatch(cgroups)

	if len(matches) == 0 {
		return "", fmt.Errorf("Cannot find container id in cgroups: %v", cgroups)
	}

	return matches[len(matches)-1], nil
}
