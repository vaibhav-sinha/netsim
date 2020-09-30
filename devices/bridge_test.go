package devices

import (
	"fmt"
	"log"
	"math/rand"
	"netsim/hardware"
	"netsim/protocol"
	"netsim/protocol/l2"
	"testing"
	"time"
)

/*
Dummy L3 Protocol implementation for testing
*/
type node struct {
	nodeNum    int
	l2Protocol protocol.Protocol
}

func (d *node) GetIdentifier() []byte {
	return []byte("no")
}

func (d *node) SendDown(data []byte, destAddr []byte, sender protocol.Protocol) {
	d.l2Protocol.SendDown(data, destAddr, d)
}

func (d *node) SendUp(b []byte) {
	log.Printf("node %d: Got packet: %s", d.nodeNum, b)
}

/*
Simple topology with 4 nodes attached to a bridge and sending data to each other
*/
func TestBridge(t *testing.T) {
	// Set the seed
	rand.Seed(time.Now().UnixNano())

	// Create the nodes
	var nodes [][]protocol.Protocol
	for i := 0; i < 4; i++ {
		var l3Protocols []protocol.Protocol
		l3Protocols = append(l3Protocols, &node{nodeNum: i})
		mac := fmt.Sprintf("portn%d", i)
		adapter := hardware.NewEthernetAdapter([]byte(mac), false)
		l2Protocol := l2.NewSimpleEthernet(adapter, l3Protocols, nil)
		l3Protocols[0].(*node).l2Protocol = l2Protocol
		nodes = append(nodes, l3Protocols)
		adapter.TurnOn()
	}

	// Create the bridge
	var macs [][]byte
	for i := 0; i < 4; i++ {
		mac := fmt.Sprintf("immac%d", i)
		macs = append(macs, []byte(mac))
	}
	bridge := NewBridge(macs)
	bridge.TurnOn()

	// Link nodes to bridge ports
	for i := 0; i < 4; i++ {
		hardware.NewSimpleDuplexLink(100, 1e8, 0.000, nodes[i][0].(*node).l2Protocol.(*l2.SimpleEthernet).GetAdapter(), bridge.GetPort(i).GetAdapter())
	}

	// Start the clock
	go hardware.Clk.Start()

	// Send traffic
	log.Printf("Testcase: Sending packet")
	nodes[0][0].(*node).SendDown([]byte("This is first packet for p2"), []byte("portn2"), nil)
	nodes[0][0].(*node).SendDown([]byte("This is second packet for p2"), []byte("portn2"), nil)

	// This should lead to a bridge forwarding table hit
	nodes[1][0].(*node).SendDown([]byte("This is first packet for p0"), []byte("portn0"), nil)
	time.Sleep(5 * time.Second)
}
