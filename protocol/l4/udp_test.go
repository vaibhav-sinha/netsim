package l4

import (
	"encoding/binary"
	"log"
	"netsim/hardware"
	"netsim/protocol"
	"netsim/protocol/l2"
	"netsim/protocol/l3"
	"testing"
	"time"
)

type node struct {
	adapter         *hardware.EthernetAdapter
	udp             *UDP
	binding         *UdpBinding
	routeProvider   protocol.RouteProvider
	addressResolver protocol.AddressResolver
}

func newNode(id int, mac []byte, ipAddr []byte) *node {
	//Create misc tcpNode components
	routeProvider := l3.NewStaticRouteProvider()
	if id == 0 {
		routeProvider.Add(protocol.DefaultRouteCidr, []byte{10, 0, 0, 2}, 0)
	} else {
		routeProvider.Add(protocol.DefaultRouteCidr, []byte{10, 0, 0, 1}, 0)
	}

	addressResolver := l3.NewStaticAddressResolver()
	if id == 0 {
		addressResolver.Add([]byte{10, 0, 0, 2}, []byte("immac2"))
	} else {
		addressResolver.Add([]byte{10, 0, 0, 1}, []byte("immac1"))
	}

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
	n.binding = n.udp.Bind(ipAddr, port, protocol.IP)
}

func (n *node) send(data []byte, ipAddr []byte, port uint16) {
	metadata := make([]byte, 4)
	binary.BigEndian.PutUint16(metadata, port)
	binary.BigEndian.PutUint16(metadata[2:4], 100)

	metadata = append(metadata, protocol.IP...)
	n.udp.SendDown(data, ipAddr, metadata, nil)
}

func (n *node) recv() {
	log.Printf("Node: Received data: %s", n.binding.Recv())
}

/*
Testcase
*/
func TestSimpleDataTransfer(t *testing.T) {
	node1 := newNode(0, []byte("immac1"), []byte{10, 0, 0, 1})
	node2 := newNode(1, []byte("immac2"), []byte{10, 0, 0, 2})

	_ = hardware.NewDuplexLink(100, 1e8, 0.00, node1.adapter, node2.adapter)

	go hardware.Clk.Start()
	node1.turnOn()
	node2.turnOn()

	//Bind tcpNode 2 to port 80
	node2.bind([]byte{0, 0, 0, 0}, 80)

	// Send the packet and wait
	log.Printf("Testcase: Sending packet")
	node1.send([]byte("this_is_a_test"), []byte{10, 0, 0, 2}, 80)
	node1.send([]byte("hope_this_works"), []byte{10, 0, 0, 2}, 80)

	node2.recv()
	node2.recv()
	time.Sleep(5 * time.Second)
}
