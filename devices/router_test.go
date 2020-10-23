package devices

import (
	"fmt"
	"log"
	"netsim/hardware"
	"netsim/protocol"
	"netsim/protocol/l2"
	"netsim/protocol/l3"
	"testing"
	"time"
)

/*
Dummy L4 Protocol implementation for testing
*/
type l4Node struct {
	l3Protocol protocol.L3Protocol
}

func newL4Node(mac []byte, ipAddr []byte, routeProvider protocol.RouteProvider, addressResolver protocol.AddressResolver) *l4Node {
	//Create the stack
	node := &l4Node{}
	adapter := hardware.NewEthernetAdapter(mac, false)
	ethernet := l2.NewEthernet(adapter, nil)
	ip := l3.NewIP([][]byte{ipAddr}, false, nil, routeProvider, addressResolver)

	//Set references
	ip.SetL2ProtocolForInterface(0, ethernet)
	node.AddL3Protocol(ip)

	//Arrange the stack
	ethernet.AddL3Protocol(ip)
	ip.AddL4Protocol(node)

	return node
}

func (d *l4Node) TurnOn() {
	d.l3Protocol.GetL2ProtocolForInterface(0).GetAdapter().TurnOn()
}

func (d *l4Node) AddL3Protocol(l3Protocol protocol.L3Protocol) {
	d.l3Protocol = l3Protocol
}

func (d *l4Node) GetL3Protocol() protocol.L3Protocol {
	return d.l3Protocol
}

func (d *l4Node) GetIdentifier() []byte {
	return []byte{6}
}

func (d *l4Node) SendDown(data []byte, destAddr []byte, metadata []byte, sender protocol.Protocol) {
	d.l3Protocol.SendDown(data, destAddr, metadata, d)
}

func (d *l4Node) SendUp(b []byte, metadata []byte, source protocol.Protocol) {
	log.Printf("l4Node: ip %v: Got packet %s", d.l3Protocol.GetAddressForInterface(0), b)
}

/*
Testcase
*/
func TestSimpleDataTransfer(t *testing.T) {
	//Create the nodes
	routeProvider1 := l3.NewStaticRouteProvider()
	routeProvider1.Add(protocol.DefaultRouteCidr, []byte{10, 0, 0, 1}, 0)

	addressResolver1 := l3.NewStaticAddressResolver()
	addressResolver1.Add([]byte{10, 0, 0, 1}, []byte("route1"))

	node1 := newL4Node([]byte("node01"), []byte{10, 0, 0, 2}, routeProvider1, addressResolver1)

	routeProvider2 := l3.NewStaticRouteProvider()
	routeProvider2.Add(protocol.DefaultRouteCidr, []byte{192, 31, 0, 1}, 0)

	addressResolver2 := l3.NewStaticAddressResolver()
	addressResolver2.Add([]byte{192, 31, 0, 1}, []byte("route2"))

	node2 := newL4Node([]byte("node02"), []byte{192, 31, 0, 2}, routeProvider2, addressResolver2)

	//Create the router
	var macs [][]byte
	for i := 1; i < 3; i++ {
		mac := fmt.Sprintf("route%d", i)
		macs = append(macs, []byte(mac))
	}

	var ipAddrs [][]byte
	ipAddrs = append(ipAddrs, []byte{10, 0, 0, 1})
	ipAddrs = append(ipAddrs, []byte{192, 31, 0, 1})

	routeProvider := l3.NewStaticRouteProvider()
	routeProvider.Add(&protocol.CIDR{Address: []byte{10, 0, 0, 0}, Mask: 24}, []byte{10, 0, 0, 2}, 0)
	routeProvider.Add(protocol.DefaultRouteCidr, []byte{192, 31, 0, 2}, 1)

	addressResolver := l3.NewStaticAddressResolver()
	addressResolver.Add([]byte{10, 0, 0, 2}, []byte("node01"))
	addressResolver.Add([]byte{192, 31, 0, 2}, []byte("node02"))

	router := NewRouter(macs, ipAddrs, routeProvider, addressResolver)

	//Link the hardware
	_ = hardware.NewDuplexLink(100, 1e8, 0.00, node1.GetL3Protocol().GetL2ProtocolForInterface(0).GetAdapter(), router.GetL3Protocol().GetL2ProtocolForInterface(0).GetAdapter())
	_ = hardware.NewDuplexLink(100, 1e8, 0.00, node2.GetL3Protocol().GetL2ProtocolForInterface(0).GetAdapter(), router.GetL3Protocol().GetL2ProtocolForInterface(1).GetAdapter())

	//Start everything
	go hardware.Clk.Start()
	node1.TurnOn()
	node2.TurnOn()
	router.TurnOn()

	// Send the packet and wait
	log.Printf("Testcase: Sending packet")
	node1.SendDown([]byte("this_is_a_test"), []byte{192, 31, 0, 2}, []byte{0, 5}, nil)
	node1.SendDown([]byte("hope_this_works"), []byte{192, 31, 0, 2}, []byte{0, 5}, nil)

	// Send large packet for fragmentation - Set Ethernet MTU to 35: 20 bytes header and 15 bytes payload
	// Sending body greater than 15 bytes should cause fragmentation
	node2.SendDown([]byte("this_is_a_test_and_it_should_cause_fragmentation"), []byte{10, 0, 0, 2}, []byte{0, 5}, nil)
	time.Sleep(10 * time.Second)

}
