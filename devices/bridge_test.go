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
type l3Node struct {
	nodeNum    int
	l2Protocol protocol.L2Protocol
}

func NewL3Node(mac []byte, nodeNum int) *l3Node {
	node := &l3Node{
		nodeNum: nodeNum,
	}
	adapter := hardware.NewEthernetAdapter(mac, false)
	ethernet := l2.NewEthernet(adapter, nil)
	ethernet.AddL3Protocol(node)
	node.l2Protocol = ethernet
	return node
}

func (d *l3Node) TurnOn() {
	d.l2Protocol.GetAdapter().TurnOn()
}

func (d *l3Node) GetIdentifier() []byte {
	return []byte("no")
}

func (d *l3Node) GetAddressForInterface(intfNum int) []byte {
	return []byte{10, 0, 0, 1}
}

func (d *l3Node) SetL2ProtocolForInterface(intfNum int, l2Protocol protocol.L2Protocol) {
	d.l2Protocol = l2Protocol
}

func (d *l3Node) GetL2ProtocolForInterface(intfNum int) protocol.L2Protocol {
	return d.l2Protocol
}

func (d *l3Node) AddL4Protocol(l4Protocol protocol.L4Protocol) {

}

func (d *l3Node) SendDown(data []byte, destAddr []byte, metadata []byte, sender protocol.Protocol) {
	d.l2Protocol.SendDown(data, destAddr, metadata, d)
}

func (d *l3Node) SendUp(b []byte, metadata []byte, sender protocol.Protocol) {
	log.Printf("l4Node %d: Got packet: %s", d.nodeNum, b)
}

/*
Simple topology with 4 nodes attached to a bridge and sending data to each other
*/
func TestBridge(t *testing.T) {
	// Set the seed
	rand.Seed(time.Now().UnixNano())

	// Create the nodes
	nodes := []*l3Node{}
	for i := 0; i < 4; i++ {
		mac := fmt.Sprintf("portn%d", i)
		n := NewL3Node([]byte(mac), i)
		nodes = append(nodes, n)
		n.TurnOn()
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
		hardware.NewDuplexLink(100, 1e8, 0.000, nodes[i].l2Protocol.GetAdapter(), bridge.GetPort(i).GetAdapter())
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
