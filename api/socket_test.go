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
	tcp             *l4.TCP
	routeProvider   protocol.RouteProvider
	addressResolver protocol.AddressResolver
}

func newNode(id int, mac []byte, ipAddr []byte) *node {
	//Create misc node components
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
	udp := l4.NewUDP()
	tcp := l4.NewTCP()

	//Set references
	ip.SetL2ProtocolForInterface(0, ethernet)

	//Arrange the stack
	ethernet.AddL3Protocol(ip)
	ip.AddL4Protocol(udp)
	ip.AddL4Protocol(tcp)
	udp.AddL3Protocol(ip)
	tcp.AddL3Protocol(ip)

	n.udp = udp
	n.tcp = tcp
	return n
}

func (n *node) turnOn() {
	n.adapter.TurnOn()
}

func (n *node) GetUDP() *l4.UDP {
	return n.udp
}

func (n *node) GetTCP() *l4.TCP {
	return n.tcp
}

/*
Testcase
*/
func TestUDP(t *testing.T) {
	node1 := newNode(0, []byte("immac1"), []byte{10, 0, 0, 1})
	node2 := newNode(1, []byte("immac2"), []byte{10, 0, 0, 2})

	_ = hardware.NewDuplexLink(100, 1e8, 0.00, node1.adapter, node2.adapter)

	go hardware.Clk.Start()
	node1.turnOn()
	node2.turnOn()

	//Create sockets for communication
	socket1 := NewSocket(node1, AF_INET, SOCK_DGRAM, 0)
	socket2 := NewSocket(node2, AF_INET, SOCK_DGRAM, 0)

	go func(sock *Socket) {
		//Bind node 2 to port 80
		socket2.Bind([]byte{0, 0, 0, 0}, 80)

		for {
			data := sock.Recv(10)
			if data != nil {
				log.Printf("Received: %s", data)
			}
		}
	}(socket2)

	time.Sleep(1 * time.Second)

	go func(sock *Socket) {
		sock.SendTo([]byte{10, 0, 0, 2}, 80, nil, []byte("this_is_a_test"))
		sock.SendTo([]byte{10, 0, 0, 2}, 80, nil, []byte("hope_this_works"))
	}(socket1)

	time.Sleep(5 * time.Second)
	socket2.Close()
}

func TestTCP(t *testing.T) {
	node1 := newNode(0, []byte("immac1"), []byte{10, 0, 0, 1})
	node2 := newNode(1, []byte("immac2"), []byte{10, 0, 0, 2})

	_ = hardware.NewDuplexLink(100, 1e8, 0.00, node1.adapter, node2.adapter)

	go hardware.Clk.Start()
	node1.turnOn()
	node2.turnOn()

	//Create sockets for communication
	socket1 := NewSocket(node1, AF_INET, SOCK_STREAM, 0)
	socket2 := NewSocket(node2, AF_INET, SOCK_STREAM, 0)

	go func(server *Socket) {
		//Bind node 2 to port 80
		server.Bind([]byte{0, 0, 0, 0}, 80)

		//List on the port
		server.Listen(10)

		//Accept a connection
		sock := server.Accept()

		//Receive data
		for {
			data := sock.Recv(100)
			if len(data) > 0 {
				log.Printf("Recv: %s", data)
			}
		}
	}(socket2)

	time.Sleep(1 * time.Second)

	go func(client *Socket) {
		//Connect to tcpNode 2 port 80
		client.Connect([]byte{10, 0, 0, 2}, 80)

		//Send some data
		client.Send([]byte("this_is_a_test"))
		time.Sleep(1 * time.Second)
		client.Send([]byte("hope_this_works"))

		//Close the connection
		time.Sleep(5 * time.Second)
		client.Close()
	}(socket1)

	time.Sleep(10 * time.Second)
}
