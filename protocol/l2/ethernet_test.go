package l2

import (
	"log"
	"netsim/hardware"
	"netsim/protocol"
	"testing"
	"time"
)

/*
Dummy L3 Protocol implementation for testing
*/
type node struct {
	l2Protocol protocol.L2Protocol
}

func (d *node) SetL2ProtocolForInterface(intfNum int, l2Protocol protocol.L2Protocol) {
	d.l2Protocol = l2Protocol
}

func (d *node) GetL2ProtocolForInterface(intfNum int) protocol.L2Protocol {
	return d.l2Protocol
}

func (d *node) AddL4Protocol(protocol protocol.L4Protocol) {

}

func (d *node) GetIdentifier() []byte {
	return []byte("du")
}

func (d *node) GetAddressForInterface(intfNum int) []byte {
	return []byte{10, 0, 0, 1}
}

func (d *node) SendDown(data []byte, destAddr []byte, metadata []byte, sender protocol.Protocol) {
	d.l2Protocol.SendDown(data, d.getMacForAddr(destAddr), metadata, d)
}

func (d *node) SendUp(b []byte, metadata []byte, source protocol.Protocol) {
	log.Printf("node: Got packet %s", b)
}

func (d *node) getMacForAddr(destAddr []byte) []byte {
	return []byte("immac2")
}

/*
Testcase
*/
func TestSimpleDataTransfer(t *testing.T) {
	node1 := &node{}
	node2 := &node{}

	adapter1 := hardware.NewEthernetAdapter([]byte("immac1"), false)
	ethernet1 := NewEthernet(adapter1, nil)
	ethernet1.AddL3Protocol(node1)
	node1.SetL2ProtocolForInterface(0, ethernet1)

	adapter2 := hardware.NewEthernetAdapter([]byte("immac2"), false)
	ethernet2 := NewEthernet(adapter2, nil)
	ethernet2.AddL3Protocol(node2)
	node2.SetL2ProtocolForInterface(0, ethernet2)

	_ = hardware.NewLink(100, 1e8, 0.01, adapter1, adapter2)

	go hardware.Clk.Start()
	adapter1.TurnOn()
	adapter2.TurnOn()

	// Send the packet and wait
	log.Printf("Testcase: Sending packet")
	node1.SendDown([]byte("this_is_a_test"), []byte("10.0.1.1"), nil, nil)
	node1.SendDown([]byte("hope_this_works"), []byte("10.0.1.1"), nil, nil)
	time.Sleep(5 * time.Second)

}
