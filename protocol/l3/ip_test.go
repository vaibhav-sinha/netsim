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

func newNode(mac []byte, ipAddr []byte, routeProvider protocol.RouteProvider, addressResolver protocol.AddressResolver) *node {
	//Create the stack
	n := &node{}
	adapter := hardware.NewEthernetAdapter(mac, false)
	ethernet := l2.NewEthernet(adapter, nil)
	ip := NewIP([][]byte{ipAddr}, false, nil, routeProvider, addressResolver)

	//Set references
	ip.SetL2ProtocolForInterface(0, ethernet)
	n.AddL3Protocol(ip)

	//Arrange the stack
	ethernet.AddL3Protocol(ip)
	ip.AddL4Protocol(n)
	return n
}

func (d *node) AddL3Protocol(l3Protocol protocol.L3Protocol) {
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

	node1 := newNode([]byte("immac1"), []byte{10, 0, 0, 1}, routeProvider, addressResolver)
	node2 := newNode([]byte("immac2"), []byte{10, 0, 0, 2}, routeProvider, addressResolver)

	_ = hardware.NewLink(100, 1e8, 0.00, node1.l3Protocol.GetL2ProtocolForInterface(0).GetAdapter(), node2.l3Protocol.GetL2ProtocolForInterface(0).GetAdapter())

	go hardware.Clk.Start()
	node1.l3Protocol.GetL2ProtocolForInterface(0).GetAdapter().TurnOn()
	node2.l3Protocol.GetL2ProtocolForInterface(0).GetAdapter().TurnOn()

	// Send the packet and wait
	log.Printf("Testcase: Sending packet")
	node1.SendDown([]byte("this_is_a_test"), []byte{10, 0, 0, 2}, []byte{0, 5}, nil)
	node1.SendDown([]byte("hope_this_works"), []byte{10, 0, 0, 2}, []byte{0, 5}, nil)

	// Send large packet for fragmentation - Set Ethernet MTU to 35: 20 bytes header and 15 bytes payload
	// Sending body greater than 15 bytes should cause fragmentation
	node1.SendDown([]byte("this_is_a_test_and_it_should_cause_fragmentation"), []byte{10, 0, 0, 2}, []byte{0, 5}, nil)
	time.Sleep(10 * time.Second)
}
