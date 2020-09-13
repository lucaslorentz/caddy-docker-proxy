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
		t.Fatalf("Could not extract container id : %v", err)
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
		t.Fatalf("Could not extract container id : %v", err)
	}

	if actual != expected {
		t.Fatalf("id mismatch: actual %v, expected %v", actual, expected)
	}
}

func TestExtractSlashPrefixedDockerId(t *testing.T) {
	read :=
		`6:blkio:/system.slice/docker/d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442b
		5:cpuset:/system.slice/docker/d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442b
		4:net_cls,net_prio:/system.slice/docker/d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442b
		3:freezer:/system.slice/docker/d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442b
		2:cpu,cpuacct:/system.slice/docker/d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442b
		`
	expected := "d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442b"

	utils := dockerUtils{}

	actual, err := utils.ExtractContainerID(read)

	if err != nil {
		t.Fatalf("Could not extract container id : %v", err)
	}

	if actual != expected {
		t.Fatalf("id mismatch: actual %v, expected %v", actual, expected)
	}
}

func TestExtractNestedDockerId(t *testing.T) {
	read :=
		`11:devices:/docker/c59fb9264a25958577e23808e80d82acd5a27a3758e2095d1607df134221fae3/docker/ac5c7f517c5707de3c77f33f0fa43e9c625d64d1e4d6c9c8ce7b50339ec86f61
	10:perf_event:/docker/c59fb9264a25958577e23808e80d82acd5a27a3758e2095d1607df134221fae3/docker/ac5c7f517c5707de3c77f33f0fa43e9c625d64d1e4d6c9c8ce7b50339ec86f61
	9:pids:/docker/c59fb9264a25958577e23808e80d82acd5a27a3758e2095d1607df134221fae3/docker/ac5c7f517c5707de3c77f33f0fa43e9c625d64d1e4d6c9c8ce7b50339ec86f61
	8:cpuset:/docker/c59fb9264a25958577e23808e80d82acd5a27a3758e2095d1607df134221fae3/docker/ac5c7f517c5707de3c77f33f0fa43e9c625d64d1e4d6c9c8ce7b50339ec86f61
	7:hugetlb:/docker/c59fb9264a25958577e23808e80d82acd5a27a3758e2095d1607df134221fae3/docker/ac5c7f517c5707de3c77f33f0fa43e9c625d64d1e4d6c9c8ce7b50339ec86f61
	6:freezer:/docker/c59fb9264a25958577e23808e80d82acd5a27a3758e2095d1607df134221fae3/docker/ac5c7f517c5707de3c77f33f0fa43e9c625d64d1e4d6c9c8ce7b50339ec86f61
	5:net_cls,net_prio:/docker/c59fb9264a25958577e23808e80d82acd5a27a3758e2095d1607df134221fae3/docker/ac5c7f517c5707de3c77f33f0fa43e9c625d64d1e4d6c9c8ce7b50339ec86f61
	4:cpu,cpuacct:/docker/c59fb9264a25958577e23808e80d82acd5a27a3758e2095d1607df134221fae3/docker/ac5c7f517c5707de3c77f33f0fa43e9c625d64d1e4d6c9c8ce7b50339ec86f61
	3:blkio:/docker/c59fb9264a25958577e23808e80d82acd5a27a3758e2095d1607df134221fae3/docker/ac5c7f517c5707de3c77f33f0fa43e9c625d64d1e4d6c9c8ce7b50339ec86f61
	2:memory:/docker/c59fb9264a25958577e23808e80d82acd5a27a3758e2095d1607df134221fae3/docker/ac5c7f517c5707de3c77f33f0fa43e9c625d64d1e4d6c9c8ce7b50339ec86f61
	1:name=systemd:/docker/c59fb9264a25958577e23808e80d82acd5a27a3758e2095d1607df134221fae3/docker/ac5c7f517c5707de3c77f33f0fa43e9c625d64d1e4d6c9c8ce7b50339ec86f61`

	expected := "ac5c7f517c5707de3c77f33f0fa43e9c625d64d1e4d6c9c8ce7b50339ec86f61"

	utils := dockerUtils{}

	actual, err := utils.ExtractContainerID(read)

	if err != nil {
		t.Fatalf("Could not extract container id : %v", err)
	}

	if actual != expected {
		t.Fatalf("id mismatch: actual %v, expected %v", actual, expected)
	}
}

func TestExtractAKSDockerId(t *testing.T) {
	read :=
		`12:perf_event:/kubepods/pod54ebaa4a-f470-11ea-b463-000d3a9ecdb6/43172aa658cbf50b2e646e3aa4c90447b10774d4e76ff720b0f4faebdb759857
	11:cpuset:/kubepods/pod54ebaa4a-f470-11ea-b463-000d3a9ecdb6/43172aa658cbf50b2e646e3aa4c90447b10774d4e76ff720b0f4faebdb759857
	10:memory:/kubepods/pod54ebaa4a-f470-11ea-b463-000d3a9ecdb6/43172aa658cbf50b2e646e3aa4c90447b10774d4e76ff720b0f4faebdb759857
	9:devices:/kubepods/pod54ebaa4a-f470-11ea-b463-000d3a9ecdb6/43172aa658cbf50b2e646e3aa4c90447b10774d4e76ff720b0f4faebdb759857
	8:net_cls,net_prio:/kubepods/pod54ebaa4a-f470-11ea-b463-000d3a9ecdb6/43172aa658cbf50b2e646e3aa4c90447b10774d4e76ff720b0f4faebdb759857
	7:hugetlb:/kubepods/pod54ebaa4a-f470-11ea-b463-000d3a9ecdb6/43172aa658cbf50b2e646e3aa4c90447b10774d4e76ff720b0f4faebdb759857
	6:freezer:/kubepods/pod54ebaa4a-f470-11ea-b463-000d3a9ecdb6/43172aa658cbf50b2e646e3aa4c90447b10774d4e76ff720b0f4faebdb759857
	5:blkio:/kubepods/pod54ebaa4a-f470-11ea-b463-000d3a9ecdb6/43172aa658cbf50b2e646e3aa4c90447b10774d4e76ff720b0f4faebdb759857
	4:cpu,cpuacct:/kubepods/pod54ebaa4a-f470-11ea-b463-000d3a9ecdb6/43172aa658cbf50b2e646e3aa4c90447b10774d4e76ff720b0f4faebdb759857
	3:rdma:/
	2:pids:/kubepods/pod54ebaa4a-f470-11ea-b463-000d3a9ecdb6/43172aa658cbf50b2e646e3aa4c90447b10774d4e76ff720b0f4faebdb759857
	1:name=systemd:/kubepods/pod54ebaa4a-f470-11ea-b463-000d3a9ecdb6/43172aa658cbf50b2e646e3aa4c90447b10774d4e76ff720b0f4faebdb759857
	`

	expected := "43172aa658cbf50b2e646e3aa4c90447b10774d4e76ff720b0f4faebdb759857"

	utils := dockerUtils{}

	actual, err := utils.ExtractContainerID(read)

	if err != nil {
		t.Fatalf("Could not extract container id : %v", err)
	}

	if actual != expected {
		t.Fatalf("id mismatch: actual %v, expected %v", actual, expected)
	}
}

func TestExtractECSDockerId(t *testing.T) {
	read :=
		`9:perf_event:/ecs/8f67afbb-3222-488d-b96a-9262c37dc9d3/3137c30c56add55d7212fdef77fd796c69b08f7845aa9a3d3fdb720c2a885a1d
	8:memory:/ecs/8f67afbb-3222-488d-b96a-9262c37dc9d3/3137c30c56add55d7212fdef77fd796c69b08f7845aa9a3d3fdb720c2a885a1d
	7:hugetlb:/ecs/8f67afbb-3222-488d-b96a-9262c37dc9d3/3137c30c56add55d7212fdef77fd796c69b08f7845aa9a3d3fdb720c2a885a1d
	6:freezer:/ecs/8f67afbb-3222-488d-b96a-9262c37dc9d3/3137c30c56add55d7212fdef77fd796c69b08f7845aa9a3d3fdb720c2a885a1d
	5:devices:/ecs/8f67afbb-3222-488d-b96a-9262c37dc9d3/3137c30c56add55d7212fdef77fd796c69b08f7845aa9a3d3fdb720c2a885a1d
	4:cpuset:/ecs/8f67afbb-3222-488d-b96a-9262c37dc9d3/3137c30c56add55d7212fdef77fd796c69b08f7845aa9a3d3fdb720c2a885a1d
	3:cpuacct:/ecs/8f67afbb-3222-488d-b96a-9262c37dc9d3/3137c30c56add55d7212fdef77fd796c69b08f7845aa9a3d3fdb720c2a885a1d
	2:cpu:/ecs/8f67afbb-3222-488d-b96a-9262c37dc9d3/3137c30c56add55d7212fdef77fd796c69b08f7845aa9a3d3fdb720c2a885a1d
	1:blkio:/ecs/8f67afbb-3222-488d-b96a-9262c37dc9d3/3137c30c56add55d7212fdef77fd796c69b08f7845aa9a3d3fdb720c2a885a1d
	`

	expected := "3137c30c56add55d7212fdef77fd796c69b08f7845aa9a3d3fdb720c2a885a1d"

	utils := dockerUtils{}

	actual, err := utils.ExtractContainerID(read)

	if err != nil {
		t.Fatalf("Could not extract container id : %v", err)
	}

	if actual != expected {
		t.Fatalf("id mismatch: actual %v, expected %v", actual, expected)
	}
}
