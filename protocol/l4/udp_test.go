package l4

import (
	"encoding/binary"
	"log"
	"netsim/hardware"
	"netsim/protocol"
	"netsim/protocol/l2"
	"netsim/protocol/l3"
	"netsim/utils"
	"testing"
	"time"
)

type node struct {
	adapter         *hardware.EthernetAdapter
	udp             *UDP
	binding         *Binding
	routeProvider   protocol.RouteProvider
	addressResolver protocol.AddressResolver
}

func newNode(mac []byte, ipAddr []byte) *node {
	//Create misc node components
	routeProvider := &staticRouteProvider{}
	addressResolver := &staticAddressResolver{}

	//Create the stack
	n := &node{routeProvider: routeProvider, addressResolver: addressResolver}
	n.adapter = hardware.NewEthernetAdapter(mac, false)
	ethernet := l2.NewEthernet(n.adapter, nil)
	ip := l3.NewIP([][]byte{ipAddr}, false, nil, routeProvider, addressResolver)
	udp := NewUDP()

	//Set references
	ip.SetL2ProtocolForInterface(0, ethernet)

	//Arrange the stack
	ethernet.AddL3Protocol(ip)
	ip.AddL4Protocol(udp)
	udp.AddL3Protocol(ip)

	n.udp = udp
	return n
}

func (n *node) turnOn() {
	n.adapter.TurnOn()
}

func (n *node) bind(ipAddr []byte, port uint16) {
	n.binding = n.udp.Bind(ipAddr, port)
}

func (n *node) send(data []byte, ipAddr []byte, port uint16) {
	metadata := make([]byte, 4)
	binary.BigEndian.PutUint16(metadata, port)
	binary.BigEndian.PutUint16(metadata[2:4], 100)

	metadata = append(metadata, utils.HexStringToBytes("0800")...)
	n.udp.SendDown(data, ipAddr, metadata, nil)
}

func (n *node) recv() {
	log.Printf("Node: Received data: %s", n.binding.Recv())
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
	node1 := newNode([]byte("immac1"), []byte{10, 0, 0, 1})
	node2 := newNode([]byte("immac2"), []byte{10, 0, 0, 2})

	_ = hardware.NewDuplexLink(100, 1e8, 0.00, node1.adapter, node2.adapter)

	go hardware.Clk.Start()
	node1.turnOn()
	node2.turnOn()

	//Bind node 2 to port 80
	node2.bind([]byte{0, 0, 0, 0}, 80)

	// Send the packet and wait
	log.Printf("Testcase: Sending packet")
	node1.send([]byte("this_is_a_test"), []byte{10, 0, 0, 2}, 80)
	node1.send([]byte("hope_this_works"), []byte{10, 0, 0, 2}, 80)

	node2.recv()
	node2.recv()
	time.Sleep(5 * time.Second)
}
