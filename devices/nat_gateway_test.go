package devices

import (
	"fmt"
	"log"
	"netsim/api"
	"netsim/hardware"
	"netsim/protocol"
	"netsim/protocol/l3"
	"testing"
	"time"
)

/*
Testcase
*/
func TestNatDataTransfer(t *testing.T) {
	//Create the nodes
	node1 := NewComputer([]byte("node01"), []byte{10, 0, 0, 2})
	node1.AddRoute(protocol.DefaultRouteCidr, []byte{10, 0, 0, 1})
	node1.AddAddress([]byte{10, 0, 0, 1}, []byte("route1"))

	node2 := NewComputer([]byte("node02"), []byte{201, 31, 0, 2})
	node2.AddRoute(protocol.DefaultRouteCidr, []byte{201, 31, 0, 1})
	node2.AddAddress([]byte{201, 31, 0, 1}, []byte("route2"))

	//Create the NAT Gateway
	var macs [][]byte
	for i := 1; i < 3; i++ {
		mac := fmt.Sprintf("route%d", i)
		macs = append(macs, []byte(mac))
	}

	var ipAddrs [][]byte
	ipAddrs = append(ipAddrs, []byte{10, 0, 0, 1})
	ipAddrs = append(ipAddrs, []byte{201, 31, 0, 1})

	routeProvider := l3.NewStaticRouteProvider()
	routeProvider.Add(&protocol.CIDR{Address: []byte{10, 0, 0, 0}, Mask: 24}, []byte{10, 0, 0, 2}, 0)
	routeProvider.Add(protocol.DefaultRouteCidr, []byte{201, 31, 0, 2}, 1)

	addressResolver := l3.NewStaticAddressResolver()
	addressResolver.Add([]byte{10, 0, 0, 2}, []byte("node01"))
	addressResolver.Add([]byte{201, 31, 0, 2}, []byte("node02"))

	router := NewNatGateway(macs, ipAddrs, routeProvider, addressResolver)

	//Link the hardware
	_ = hardware.NewDuplexLink(100, 1e8, 0.00, node1.GetAdapter(), router.GetL3Protocol().GetL2ProtocolForInterface(0).GetAdapter())
	_ = hardware.NewDuplexLink(100, 1e8, 0.00, node2.GetAdapter(), router.GetL3Protocol().GetL2ProtocolForInterface(1).GetAdapter())

	//Start everything
	go hardware.Clk.Start()
	node1.TurnOn()
	node2.TurnOn()
	router.TurnOn()

	log.Printf("Testcase: Connecting and Sending packet")
	node2.Run(server)
	node1.Run(client)

	time.Sleep(20 * time.Second)
}

func server(server *Computer) {
	socket := server.NewSocket(api.AF_INET, api.SOCK_STREAM, 0)

	//Bind to port 80
	socket.Bind([]byte{0, 0, 0, 0}, 80)

	//List on the port
	socket.Listen(10)

	//Accept a connection
	sock := socket.Accept()

	//Receive data
	for {
		data := sock.Recv(100)
		if len(data) > 0 {
			log.Printf("Server Received: %s", data)
		}
	}
}

func client(client *Computer) {
	socket := client.NewSocket(api.AF_INET, api.SOCK_STREAM, 0)

	//Connect to the server
	socket.Connect([]byte{201, 31, 0, 2}, 80)

	//Send some data
	socket.Send([]byte("Hello"))

	//Close the connection
	time.Sleep(10 * time.Second)
	socket.Close()
}
