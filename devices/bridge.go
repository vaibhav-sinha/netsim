package devices

import (
	"netsim/hardware"
	"netsim/protocol/l2"
	"sync"
)

/*
A bridge is an ethernet switch.
Ideally, each entry in the forwardingTable should have an expiry but since it would not be useful in simulations, hence
not implementing it.
Also, not adding any buffers
*/
type Bridge struct {
	ports           []*l2.SimpleEthernet
	forwardingTable map[string]int
	lock            sync.Mutex
}

func NewBridge(macs [][]byte) *Bridge {
	bridge := &Bridge{
		forwardingTable: make(map[string]int),
	}
	for i, m := range macs {
		adapter := l2.NewSimpleEthernet(hardware.NewEthernetAdapter(m, true), nil, newBridgeFrameConsumer(bridge, i))
		bridge.ports = append(bridge.ports, adapter)
	}

	return bridge
}

func (b *Bridge) TurnOn() {
	for _, a := range b.ports {
		a.GetAdapter().TurnOn()
	}
}

func (b *Bridge) GetPort(portNum int) *l2.SimpleEthernet {
	return b.ports[portNum]
}

/*
Actual forwarding logic
*/
func (b *Bridge) sendUp(portNum int, frame []byte) {
	b.lock.Lock()
	defer b.lock.Unlock()

	sourceAddr := frame[14:20]
	destAddr := frame[8:14]

	//Make an entry in the forwarding table for the source address
	b.forwardingTable[string(sourceAddr)] = portNum

	//If an entry for the destination exists in the forwarding table then forward the frame there, else everywhere
	destPortNum, ok := b.forwardingTable[string(destAddr)]
	if ok {
		b.ports[destPortNum].GetAdapter().PutInBuffer(frame)
	} else {
		for i, port := range b.ports {
			if i != portNum {
				port.GetAdapter().PutInBuffer(frame)
			}
		}
	}
}

/*
Internal struct to track which port sent a frame
*/
type bridgeFrameConsumer struct {
	bridge  *Bridge
	portNum int
}

func newBridgeFrameConsumer(bridge *Bridge, portNum int) *bridgeFrameConsumer {
	return &bridgeFrameConsumer{
		bridge:  bridge,
		portNum: portNum,
	}
}

func (b *bridgeFrameConsumer) SendUp(frame []byte) {
	b.bridge.sendUp(b.portNum, frame)
}
