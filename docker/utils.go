package docker

import (
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
	var containerID string
	var err error
	if runtime.GOOS == "linux" {
		if containerID == "" && err == nil {
			containerID, err = wrapper.getCurrentContainerIDFromCGroup()
		}
		if containerID == "" && err == nil {
			containerID, err = wrapper.getCurrentContainerIDFromMountInfo()
		}
	}
	if containerID == "" && err == nil {
		containerID, err = os.Hostname()
	}
	return containerID, err
}

func (wrapper *dockerUtils) getCurrentContainerIDFromMountInfo() (string, error) {
	bytes, err := ioutil.ReadFile("/proc/self/mountinfo")
	if err != nil {
		return "", err
	}
	containerID := wrapper.extractContainerIDFromMountInfo(string(bytes))
	return containerID, nil
}

func (wrapper *dockerUtils) getCurrentContainerIDFromCGroup() (string, error) {
	bytes, err := ioutil.ReadFile("/proc/self/cgroup")
	if err != nil {
		return "", err
	}
	containerID := wrapper.extractContainerIDFromCGroups(string(bytes))
	return containerID, nil
}

func (wrapper *dockerUtils) extractContainerIDFromMountInfo(cgroups string) string {
	idRegex := regexp.MustCompile(`containers/([[:alnum:]]{64})/`)
	matches := idRegex.FindStringSubmatch(cgroups)
	if len(matches) == 0 {
		return ""
	}
	return matches[len(matches)-1]
}

func (wrapper *dockerUtils) extractContainerIDFromCGroups(cgroups string) string {
	idRegex := regexp.MustCompile(`(?im)^[^:]*:[^:]*:.*\b([[:alnum:]]{64})\b`)
	matches := idRegex.FindStringSubmatch(cgroups)
	if len(matches) == 0 {
		return ""
	}
	return matches[len(matches)-1]
}
