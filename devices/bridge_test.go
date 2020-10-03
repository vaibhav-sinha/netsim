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
	l2Protocol protocol.L2Protocol
}

func (d *node) GetIdentifier() []byte {
	return []byte("no")
}

func (d *node) SetL2Protocol(l2Protocol protocol.L2Protocol) {
	d.l2Protocol = l2Protocol
}

func (d *node) SendDown(data []byte, destAddr []byte, metadata []byte, sender protocol.Protocol) {
	d.l2Protocol.SendDown(data, destAddr, metadata, d)
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
	nodes := []*node{}
	for i := 0; i < 4; i++ {
		var l3Protocols []protocol.L3Protocol
		n := &node{nodeNum: i}
		l3Protocols = append(l3Protocols, n)
		mac := fmt.Sprintf("portn%d", i)
		adapter := hardware.NewEthernetAdapter([]byte(mac), false)
		l2.NewSimpleEthernet(adapter, l3Protocols, nil)
		nodes = append(nodes, n)
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
		hardware.NewSimpleDuplexLink(100, 1e8, 0.000, nodes[i].l2Protocol.GetAdapter(), bridge.GetPort(i).GetAdapter())
	}

	// Start the clock
	go hardware.Clk.Start()

	// Send traffic
	log.Printf("Testcase: Sending packet")
	nodes[0].SendDown([]byte("This is first packet for p2"), []byte("portn2"), nil, nil)
	nodes[0].SendDown([]byte("This is second packet for p2"), []byte("portn2"), nil, nil)

	// This should lead to a bridge forwarding table hit
	nodes[2].SendDown([]byte("This is first packet for p0"), []byte("portn0"), nil, nil)

	// Setup VLAN
	bridge.AddPortToVlan(1, 1)
	nodes[1].SendDown([]byte("This packet should not reach p3"), []byte("portn3"), nil, nil)

	time.Sleep(2 * time.Second)

	// Add port 3 to VLAN 1
	bridge.AddPortToVlan(3, 1)
	nodes[1].SendDown([]byte("This packet should reach p3"), []byte("portn3"), nil, nil)

	time.Sleep(2 * time.Second)
}
