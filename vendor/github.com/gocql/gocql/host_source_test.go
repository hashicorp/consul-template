// +build all integration

package gocql

import (
	"fmt"
	"net"
	"testing"
)

func TestUnmarshalCassVersion(t *testing.T) {
	tests := [...]struct {
		data    string
		version cassVersion
	}{
		{"3.2", cassVersion{3, 2, 0}},
		{"2.10.1-SNAPSHOT", cassVersion{2, 10, 1}},
		{"1.2.3", cassVersion{1, 2, 3}},
	}

	for i, test := range tests {
		v := &cassVersion{}
		if err := v.UnmarshalCQL(nil, []byte(test.data)); err != nil {
			t.Errorf("%d: %v", i, err)
		} else if *v != test.version {
			t.Errorf("%d: expected %#+v got %#+v", i, test.version, *v)
		}
	}
}

func TestCassVersionBefore(t *testing.T) {
	tests := [...]struct {
		version             cassVersion
		major, minor, patch int
	}{
		{cassVersion{1, 0, 0}, 0, 0, 0},
		{cassVersion{0, 1, 0}, 0, 0, 0},
		{cassVersion{0, 0, 1}, 0, 0, 0},

		{cassVersion{1, 0, 0}, 0, 1, 0},
		{cassVersion{0, 1, 0}, 0, 0, 1},
	}

	for i, test := range tests {
		if !test.version.Before(test.major, test.minor, test.patch) {
			t.Errorf("%d: expected v%d.%d.%d to be before %v", i, test.major, test.minor, test.patch, test.version)
		}
	}

}

func TestIsValidPeer(t *testing.T) {
	ring := ringDescriber{}
	host := &HostInfo{
		rpcAddress: net.ParseIP("0.0.0.0"),
		rack:       "myRack",
		hostId:     "0",
		dataCenter: "datacenter",
		tokens:     []string{"0", "1"},
	}

	if !ring.IsValidPeer(host) {
		t.Errorf("expected %+v to be a valid peer", host)
	}

	host.rack = ""
	if ring.IsValidPeer(host) {
		t.Errorf("expected %+v to NOT be a valid peer", host)
	}
}

func TestGetHosts(t *testing.T) {
	cluster := createCluster()
	session := createSessionFromCluster(cluster, t)

	hosts, partitioner, err := session.hostSource.GetHosts()

	assertTrue(t, "err == nil", err == nil)
	assertTrue(t, "len(hosts) == 3", len(hosts) == 3)
	assertTrue(t, "len(partitioner) != 0", len(partitioner) != 0)

}

func TestGetHostsWithFilter(t *testing.T) {
	filterHostIP := net.ParseIP("127.0.0.3")
	cluster := createCluster()

	// Filter to remove one of the localhost nodes
	cluster.HostFilter = HostFilterFunc(func(host *HostInfo) bool {
		if host.ConnectAddress().Equal(filterHostIP) {
			return false
		}
		return true
	})
	session := createSessionFromCluster(cluster, t)

	hosts, partitioner, err := session.hostSource.GetHosts()
	assertTrue(t, "err == nil", err == nil)
	assertTrue(t, "len(hosts) == 2", len(hosts) == 2)
	assertTrue(t, "len(partitioner) != 0", len(partitioner) != 0)
	for _, host := range hosts {
		if host.ConnectAddress().Equal(filterHostIP) {
			t.Fatal(fmt.Sprintf("Did not expect to see '%q' in host list", filterHostIP))
		}
	}
}

func TestHostInfo_ConnectAddress(t *testing.T) {
	var localhost = net.IPv4(127, 0, 0, 1)
	tests := []struct {
		name          string
		connectAddr   net.IP
		rpcAddr       net.IP
		broadcastAddr net.IP
		peer          net.IP
	}{
		{name: "rpc_address", rpcAddr: localhost},
		{name: "connect_address", connectAddr: localhost},
		{name: "broadcast_address", broadcastAddr: localhost},
		{name: "peer", peer: localhost},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			host := &HostInfo{
				connectAddress:   test.connectAddr,
				rpcAddress:       test.rpcAddr,
				broadcastAddress: test.broadcastAddr,
				peer:             test.peer,
			}

			if addr := host.ConnectAddress(); !addr.Equal(localhost) {
				t.Fatalf("expected ConnectAddress to be %s got %s", localhost, addr)
			}
		})
	}
}
