package l4

import (
	"log"
	"netsim/hardware"
	"netsim/protocol"
	"netsim/protocol/l2"
	"netsim/protocol/l3"
	"testing"
	"time"
)

type tcpNode struct {
	adapter         *hardware.EthernetAdapter
	tcp             *TCP
	binding         *TcpBinding
	conn            *TcpConnection
	routeProvider   protocol.RouteProvider
	addressResolver protocol.AddressResolver
}

func newTcpNode(id int, mac []byte, ipAddr []byte) *tcpNode {
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
	n := &tcpNode{routeProvider: routeProvider, addressResolver: addressResolver}
	n.adapter = hardware.NewEthernetAdapter(mac, false)
	ethernet := l2.NewEthernet(n.adapter, nil)
	ip := l3.NewIP([][]byte{ipAddr}, false, nil, routeProvider, addressResolver)
	tcp := NewTCP()

	//Set references
	ip.SetL2ProtocolForInterface(0, ethernet)

	//Arrange the stack
	ethernet.AddL3Protocol(ip)
	ip.AddL4Protocol(tcp)
	tcp.AddL3Protocol(ip)

	n.tcp = tcp
	return n
}

func (n *tcpNode) turnOn() {
	n.adapter.TurnOn()
}

func (n *tcpNode) bind(ipAddr []byte, port uint16) {
	n.binding = n.tcp.Bind(ipAddr, port, protocol.IP)
}

func (n *tcpNode) listen() {
	n.binding.Listen(10)
}

func (n *tcpNode) accept() {
	n.conn = n.binding.Accept()
}

func (n *tcpNode) connect(ipAddr []byte, port uint16) {
	n.conn = n.binding.Connect(ipAddr, port)
}

func (n *tcpNode) close() {
	n.conn.Close()
}

func (n *tcpNode) send(data []byte) {
	for _, b := range data {
		n.conn.Send(b)
	}
}

func (n *tcpNode) recv() {
	b := n.conn.Recv()
	if b != nil {
		log.Printf("Node: Received data: %v", *b)
	}
}

/*
Testcase
*/
func TestSimpleReliableDataTransfer(t *testing.T) {
	node1 := newTcpNode(0, []byte("immac1"), []byte{10, 0, 0, 1})
	node2 := newTcpNode(1, []byte("immac2"), []byte{10, 0, 0, 2})

	_ = hardware.NewDuplexLink(100, 1e8, 0.00, node1.adapter, node2.adapter)

	go hardware.Clk.Start()
	node1.turnOn()
	node2.turnOn()

	// Send the packet and wait
	log.Printf("Testcase: Connecting and Sending packet")
	go runServer(node2)
	go runClient(node1)

	time.Sleep(20 * time.Second)
}

func runServer(server *tcpNode) {
	//Bind tcpNode 2 to port 80
	server.bind([]byte{0, 0, 0, 0}, 80)

	//List on the port
	server.listen()

	//Accept a connection
	server.accept()

	//Receive data
	for {
		server.recv()
	}
}

func runClient(client *tcpNode) {
	//Bind tcpNode 1 to port 8000
	client.bind([]byte{0, 0, 0, 0}, 8000)

	//Connect to tcpNode 2 port 80
	client.connect([]byte{10, 0, 0, 2}, 80)

	//Send some data
	client.send([]byte("this_is_a_test"))
	time.Sleep(1 * time.Second)
	client.send([]byte("hope_this_works"))

	//Close the connection
	time.Sleep(10 * time.Second)
	client.close()
}
