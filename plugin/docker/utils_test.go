package docker

import (
	"testing"
)

func TestFailExtractBasicDockerId(t *testing.T) {
	read :=
		`5:cpu,cpuacct:/system.slice/d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442.scope
		4:cpu,cpuacct:/system.slice/d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442
		3:zpu,cpuacct:/system.slice/d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442b
		2:cpu,cpuacct:system.slice:d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442b
		1:cpu,cpuacct:/system.slice/d39fa516d8377ecddf9bf8ef33f81cbf5 8b4d604d85293ced7cdb0c7fc52442b.scope
		`

	utils := dockerUtils{}

	actual, err := utils.ExtractContainerID(read)

	if err == nil {
		t.Fatalf("Got unexpected container id %v", actual)
	}

}

func TestExtractBasicDockerId(t *testing.T) {
	read :=
		`6:blkio:/system.slice/d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442b
		5:cpuset:/system.slice/d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442b
		4:net_cls,net_prio:/system.slice/d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442b
		3:freezer:/system.slice/d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442b
		2:cpu,cpuacct:/system.slice/d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442b
		`
	expected := "d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442b"

	utils := dockerUtils{}

	actual, err := utils.ExtractContainerID(read)

	if err != nil {
		t.Fatal("Could not extract container id")
	}

	if actual != expected {
		t.Fatalf("id mismatch: actual %v, expected %v", actual, expected)
	}
}

func TestExtractScopedDockerId(t *testing.T) {
	read :=
		`6:blkio:/system.slice/docker-d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442b.scope
		5:cpuset:/system.slice/docker-d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442b.scope
		4:net_cls,net_prio:/system.slice/docker-d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442b.scope
		3:freezer:/system.slice/docker-d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442b.scope
		2:cpu,cpuacct:/system.slice/docker-d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442b.scope
		`
	expected := "d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442b"

	utils := dockerUtils{}

	actual, err := utils.ExtractContainerID(read)

	if err != nil {
		t.Fatal("Could not extract container id")
	}

	if actual != expected {
		t.Fatalf("id mismatch: actual %v, expected %v", actual, expected)
	}
}

func TestExtractSlashPrefixedDockerId(t *testing.T) {
	read :=
		`6:blkio:/system.slice/docker/d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442b
		5:cpuset:/system.slice/docker/d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442b
		4:net_cls,net_prio:/system.slice/docker-d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442b
		3:freezer:/system.slice/docker/d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442b
		2:cpu,cpuacct:/system.slice/docker/d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442b
		`
	expected := "d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442b"

	utils := dockerUtils{}

	actual, err := utils.ExtractContainerID(read)

	if err != nil {
		t.Fatal("Could not extract container id")
	}

	if actual != expected {
		t.Fatalf("id mismatch: actual %v, expected %v", actual, expected)
	}
}
