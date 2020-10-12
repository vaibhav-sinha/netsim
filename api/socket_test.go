package api

import (
	"log"
	"netsim/hardware"
	"netsim/protocol"
	"netsim/protocol/l2"
	"netsim/protocol/l3"
	"netsim/protocol/l4"
	"testing"
	"time"
)

type node struct {
	adapter         *hardware.EthernetAdapter
	udp             *l4.UDP
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
	udp := l4.NewUDP()

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

func (n *node) GetUDP() *l4.UDP {
	return n.udp
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

	//Create sockets for communication
	socket1 := NewSocket(node1, AF_INET, SOCK_DGRAM, 0)
	socket2 := NewSocket(node2, AF_INET, SOCK_DGRAM, 0)

	//Bind node 2 to port 80
	socket2.Bind([]byte{0, 0, 0, 0}, 80)

	// Send the packet and wait
	log.Printf("Testcase: Sending packet")
	socket1.SendTo([]byte{10, 0, 0, 2}, 80, nil, []byte("this_is_a_test"))
	socket1.SendTo([]byte{10, 0, 0, 2}, 80, nil, []byte("hope_this_works"))

	go read(socket2)

	time.Sleep(5 * time.Second)
	socket2.Close()
}

func read(sock *Socket) {
	for {
		data := sock.Recv(10)
		if data != nil {
			log.Printf("Received: %s", data)
		}
	}
}
