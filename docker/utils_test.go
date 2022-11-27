package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtratFromMountInfo(t *testing.T) {
	read :=
		`982 811 0:188 / / rw,relatime master:211 - overlay overlay rw,lowerdir=/var/lib/docker/overlay2/l/JITAD3AQIAPDR63API26SHX5CQ:/var/lib/docker/overlay2/l/OD4G2XK3EBQC7UCYC2MKN2VU4N:/var/lib/docker/overlay2/l/XCYRLTZ7FPAFABA6UPAECHCUFM:/var/lib/docker/overlay2/l/2UTXO3KIF3I7EQFKXPOYQO6WGN,upperdir=/var/lib/docker/overlay2/11a7a30cc374c98491c15185334a99f07e2761a9759c2c5b3ba1b4122ec9fbf7/diff,workdir=/var/lib/docker/overlay2/11a7a30cc374c98491c15185334a99f07e2761a9759c2c5b3ba1b4122ec9fbf7/work
		984 982 0:205 / /proc rw,nosuid,nodev,noexec,relatime - proc proc rw
		986 982 0:207 / /dev rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
		988 986 0:209 / /dev/pts rw,nosuid,noexec,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=666
		990 982 0:211 / /sys ro,nosuid,nodev,noexec,relatime - sysfs sysfs ro
		993 990 0:33 / /sys/fs/cgroup ro,nosuid,nodev,noexec,relatime - cgroup2 cgroup rw
		994 986 0:202 / /dev/mqueue rw,nosuid,nodev,noexec,relatime - mqueue mqueue rw
		996 986 0:213 / /dev/shm rw,nosuid,nodev,noexec,relatime - tmpfs shm rw,size=65536k
		999 982 254:1 /docker/containers/d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442b/resolv.conf /etc/resolv.conf rw,relatime - ext4 /dev/vda1 rw
		1001 982 254:1 /docker/containers/d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442b/hostname /etc/hostname rw,relatime - ext4 /dev/vda1 rw
		1002 982 254:1 /docker/containers/d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442b/hosts /etc/hosts rw,relatime - ext4 /dev/vda1 rw
		1003 982 0:23 /host-services/docker.proxy.sock /run/docker.sock rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,size=608152k,mode=755
		576 984 0:205 /bus /proc/bus ro,nosuid,nodev,noexec,relatime - proc proc rw
		587 984 0:205 /fs /proc/fs ro,nosuid,nodev,noexec,relatime - proc proc rw
		588 984 0:205 /irq /proc/irq ro,nosuid,nodev,noexec,relatime - proc proc rw
		589 984 0:205 /sys /proc/sys ro,nosuid,nodev,noexec,relatime - proc proc rw
		590 984 0:205 /sysrq-trigger /proc/sysrq-trigger ro,nosuid,nodev,noexec,relatime - proc proc rw
		591 984 0:207 /null /proc/kcore rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
		592 984 0:207 /null /proc/keys rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
		593 984 0:207 /null /proc/timer_list rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
		594 990 0:216 / /sys/firmware ro,relatime - tmpfs tmpfs ro`

	expected := "d39fa516d8377ecddf9bf8ef33f81cbf58b4d604d85293ced7cdb0c7fc52442b"

	utils := dockerUtils{}

	actual := utils.extractContainerIDFromMountInfo(read)

	assert.Equal(t, expected, actual)
}

func TestFailExtractBasicDockerId(t *testing.T) {
	read :=
		`1:cpu:/not_an_id`

	utils := dockerUtils{}

	actual := utils.extractContainerIDFromCGroups(read)

	assert.Empty(t, actual)
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

	actual := utils.extractContainerIDFromCGroups(read)

	assert.Equal(t, expected, actual)
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

	actual := utils.extractContainerIDFromCGroups(read)

	assert.Equal(t, expected, actual)
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

	actual := utils.extractContainerIDFromCGroups(read)

	assert.Equal(t, expected, actual)
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

	actual := utils.extractContainerIDFromCGroups(read)

	assert.Equal(t, expected, actual)
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

	actual := utils.extractContainerIDFromCGroups(read)

	assert.Equal(t, expected, actual)
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

	actual := utils.extractContainerIDFromCGroups(read)

	assert.Equal(t, expected, actual)
}

func TestExtractRootlessDockerId(t *testing.T) {
	read :=
		`11:rdma:/
		10:freezer:/
		9:cpuset:/
		8:net_cls,net_prio:/
		7:cpu,cpuacct:/
		6:devices:/user.slice
		5:memory:/user.slice/user-1000.slice/user@1000.service
		4:perf_event:/
		3:pids:/user.slice/user-1000.slice/user@1000.service
		2:blkio:/
		1:name=systemd:/user.slice/user-1000.slice/user@1000.service/docker.service/f7df0c0b3a8d4350647486b24a5bd5785d494c1a0910cfaee66d3db0db784093
		0::/user.slice/user-1000.slice/user@1000.service/docker.service
	`

	expected := "f7df0c0b3a8d4350647486b24a5bd5785d494c1a0910cfaee66d3db0db784093"

	utils := dockerUtils{}

	actual := utils.extractContainerIDFromCGroups(read)

	assert.Equal(t, expected, actual)
}
