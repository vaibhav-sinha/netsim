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
type DummyL3Protocol struct {
	l2Protocol protocol.Protocol
}

func (d *DummyL3Protocol) GetIdentifier() []byte {
	return []byte("du")
}

func (d *DummyL3Protocol) SendDown(data []byte, destAddr []byte, sender protocol.Protocol) {
	d.l2Protocol.SendDown(data, d.getMacForAddr(destAddr), d)
}

func (d *DummyL3Protocol) SendUp(b []byte) {
	log.Printf("DummyL3Protocol: Got packet %s", b)
}

func (d *DummyL3Protocol) getMacForAddr(destAddr []byte) []byte {
	return []byte("immac2")
}

/*
Testcase
*/
func TestSimpleDataTransfer(t *testing.T) {
	var l3Protocols1 []protocol.Protocol
	l3Protocols1 = append(l3Protocols1, &DummyL3Protocol{})

	var l3Protocols2 []protocol.Protocol
	l3Protocols2 = append(l3Protocols2, &DummyL3Protocol{})

	adapter1 := hardware.NewEthernetAdapter([]byte("immac1"), false)
	l2Protocol1 := NewSimpleEthernet(adapter1, l3Protocols1)
	l3Protocols1[0].(*DummyL3Protocol).l2Protocol = l2Protocol1

	adapter2 := hardware.NewEthernetAdapter([]byte("immac2"), false)
	l2Protocol2 := NewSimpleEthernet(adapter2, l3Protocols2)
	l3Protocols2[0].(*DummyL3Protocol).l2Protocol = l2Protocol2

	_ = hardware.NewSimpleLink(100, 1e8, 0.01, adapter1, adapter2)

	go hardware.Clk.Start()
	adapter1.TurnOn()
	adapter2.TurnOn()

	// Send the packet and wait
	log.Printf("Testcase: Sending packet")
	l3Protocols1[0].SendDown([]byte("this_is_a_test"), []byte("10.0.1.1"), nil)
	l3Protocols1[0].SendDown([]byte("hope_this_works"), []byte("10.0.1.1"), nil)
	time.Sleep(5 * time.Second)

}
