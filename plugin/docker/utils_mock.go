package docker

// UtilsMock allows mocking docker Utils
type UtilsMock struct {
	MockGetCurrentContainerID func() (string, error)
}

// GetCurrentContainerID returns the id of the container running this application
func (mock *UtilsMock) GetCurrentContainerID() (string, error) {
	return mock.MockGetCurrentContainerID()
}
