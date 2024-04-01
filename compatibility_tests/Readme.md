# Run Consul Template Compatibility Tests

## Prerequisites

### Check whether the cgroup2 filesystem is mounted with following mount configuration:
```mount | grep cgroup```

### If not, mount the cgroup2 filesystem with the following command:
```sudo mount -t cgroup2 -o rw,nosuid,nodev,noexec,relatime,memory_recursiveprot none /sys/fs/cgroup```

## How to run the compatibility tests
1. Update the versions of Consul and tag of Consul Template in Makefile. For ENT builds, set an env variable `CONSUL_LICENSE` before running the commands.
2. Run the following command:
```shell
cd compatibility_tests
make test-compat
```
3. The test results will be displayed on the console.
4. If the tests fail, check the container for the logs.
5. To clean up the containers, run the following command:
```shell
make clean
```
