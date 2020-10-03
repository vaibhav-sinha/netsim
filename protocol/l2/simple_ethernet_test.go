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
type dummyL3Protocol struct {
	l2Protocol protocol.L2Protocol
}

func (d *dummyL3Protocol) SetL2Protocol(l2Protocol protocol.L2Protocol) {
	d.l2Protocol = l2Protocol
}

func (d *dummyL3Protocol) GetIdentifier() []byte {
	return []byte("du")
}

func (d *dummyL3Protocol) SendDown(data []byte, destAddr []byte, metadata []byte, sender protocol.Protocol) {
	d.l2Protocol.SendDown(data, d.getMacForAddr(destAddr), metadata, d)
}

func (d *dummyL3Protocol) SendUp(b []byte) {
	log.Printf("dummyL3Protocol: Got packet %s", b)
}

func (d *dummyL3Protocol) getMacForAddr(destAddr []byte) []byte {
	return []byte("immac2")
}

/*
Testcase
*/
func TestSimpleDataTransfer(t *testing.T) {
	var l3Protocols1 []protocol.L3Protocol
	l3Protocols1 = append(l3Protocols1, &dummyL3Protocol{})

	var l3Protocols2 []protocol.L3Protocol
	l3Protocols2 = append(l3Protocols2, &dummyL3Protocol{})

	adapter1 := hardware.NewEthernetAdapter([]byte("immac1"), false)
	NewSimpleEthernet(adapter1, l3Protocols1, nil)

	adapter2 := hardware.NewEthernetAdapter([]byte("immac2"), false)
	NewSimpleEthernet(adapter2, l3Protocols2, nil)

	_ = hardware.NewSimpleLink(100, 1e8, 0.01, adapter1, adapter2)

	go hardware.Clk.Start()
	adapter1.TurnOn()
	adapter2.TurnOn()

	// Send the packet and wait
	log.Printf("Testcase: Sending packet")
	l3Protocols1[0].SendDown([]byte("this_is_a_test"), []byte("10.0.1.1"), nil, nil)
	l3Protocols1[0].SendDown([]byte("hope_this_works"), []byte("10.0.1.1"), nil, nil)
	time.Sleep(5 * time.Second)

}
