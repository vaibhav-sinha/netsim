package l3

import (
	"log"
	"netsim/hardware"
	"netsim/protocol"
	"netsim/protocol/l2"
	"testing"
	"time"
)

/*
Dummy L4 Protocol implementation for testing
*/
type node struct {
	l3Protocol protocol.L3Protocol
}

func (d *node) SetL3Protocol(l3Protocol protocol.L3Protocol) {
	d.l3Protocol = l3Protocol
}

func (d *node) GetL3Protocol() protocol.L3Protocol {
	return d.l3Protocol
}

func (d *node) GetIdentifier() []byte {
	return []byte("d")
}

func (d *node) SendDown(data []byte, destAddr []byte, metadata []byte, sender protocol.Protocol) {
	d.l3Protocol.SendDown(data, destAddr, metadata, d)
}

func (d *node) SendUp(b []byte, metadata []byte, source protocol.Protocol) {
	log.Printf("node: Got packet %s", b)
}

/*
Static Routing Table
*/
type staticRouteProvider struct {
}

func (s *staticRouteProvider) GetGatewayForAddress(ipAddr []byte) []byte {
	return ipAddr
}

func (s *staticRouteProvider) GetInterfaceForAddress(ipAddr []byte) int {
	return 0
}

/*
Static Address Resolution
*/
type staticAddressResolver struct {
}

func (s *staticAddressResolver) Resolve(ipAddr []byte) []byte {
	return []byte("immac2")
}

/*
Testcase
*/
func TestSimpleDataTransfer(t *testing.T) {
	routeProvider := &staticRouteProvider{}
	addressResolver := &staticAddressResolver{}

	node1 := &node{}
	node2 := &node{}

	ip1 := NewIP([]byte{10, 0, 0, 1}, false, []protocol.L4Protocol{node1}, nil, routeProvider, addressResolver)
	adapter1 := hardware.NewEthernetAdapter([]byte("immac1"), false)
	l2.NewEthernet(adapter1, []protocol.L3Protocol{ip1}, nil)

	ip2 := NewIP([]byte{10, 0, 0, 2}, false, []protocol.L4Protocol{node2}, nil, routeProvider, addressResolver)
	adapter2 := hardware.NewEthernetAdapter([]byte("immac2"), false)
	l2.NewEthernet(adapter2, []protocol.L3Protocol{ip2}, nil)

	_ = hardware.NewLink(100, 1e8, 0.00, adapter1, adapter2)

	go hardware.Clk.Start()
	adapter1.TurnOn()
	adapter2.TurnOn()

	// Send the packet and wait
	log.Printf("Testcase: Sending packet")
	node1.SendDown([]byte("this_is_a_test"), []byte{10, 0, 0, 2}, []byte{0, 5}, nil)
	node1.SendDown([]byte("hope_this_works"), []byte{10, 0, 0, 2}, []byte{0, 5}, nil)

	// Send large packet for fragmentation - Set Ethernet MTU to 35: 20 bytes header and 15 bytes payload
	// Sending body greater than 15 bytes should cause fragmentation
	node1.SendDown([]byte("this_is_a_test_and_it_should_cause_fragmentation"), []byte{10, 0, 0, 2}, []byte{0, 5}, nil)
	time.Sleep(10 * time.Second)

}
