package devices

import (
	"encoding/binary"
	"netsim/hardware"
	"netsim/protocol"
	"netsim/protocol/l2"
	"netsim/utils"
	"sync"
)

/*
A bridge is an ethernet switch.
Ideally, each entry in the forwardingTable should have an expiry but since it would not be useful in simulations, hence
not implementing it.
Also, not adding any buffers
*/
type Bridge struct {
	ports           []*l2.Ethernet
	portMapping     map[protocol.FrameConsumer]int
	forwardingTable map[string]int
	vlanTable       map[int][]uint16
	lock            sync.Mutex
}

func NewBridge(macs [][]byte) *Bridge {
	bridge := &Bridge{
		portMapping:     make(map[protocol.FrameConsumer]int),
		forwardingTable: make(map[string]int),
		vlanTable:       make(map[int][]uint16),
	}
	for i, m := range macs {
		ethernet := l2.NewEthernet(hardware.NewEthernetAdapter(m, true), bridge)
		bridge.ports = append(bridge.ports, ethernet)
		bridge.vlanTable[i] = []uint16{0}
		bridge.portMapping[ethernet] = i
	}

	return bridge
}

func (b *Bridge) AddPortToVlan(portNum int, vlanId uint16) {
	b.vlanTable[portNum] = append(b.vlanTable[portNum], vlanId)
}

func (b *Bridge) TurnOn() {
	for _, a := range b.ports {
		a.GetAdapter().TurnOn()
	}
}

func (b *Bridge) TurnOff() {
	for _, a := range b.ports {
		a.GetAdapter().TurnOff()
	}
}

func (b *Bridge) GetPort(portNum int) *l2.Ethernet {
	return b.ports[portNum]
}

/*
Actual forwarding logic
*/
func (b *Bridge) SendUp(frame []byte, metadata []byte, sender protocol.Protocol) {
	b.lock.Lock()
	defer b.lock.Unlock()

	sourceAddr := frame[14:20]
	destAddr := frame[8:14]

	portNum := b.portMapping[sender]

	//Find the VLAN Id for the incoming frame
	isSourceTrunk := b.isTrunk(portNum)
	var vlanId uint16
	if isSourceTrunk {
		vlanId = binary.BigEndian.Uint16(frame[20:22])
	} else {
		vlanId = b.getVlanId(portNum)
	}

	//Tag the frame
	if !isSourceTrunk {
		vlanTag := make([]byte, 2)
		binary.BigEndian.PutUint16(vlanTag, vlanId)
		frame[20] = vlanTag[0]
		frame[21] = vlanTag[1]

		checksum := utils.CalculateChecksum(frame[:len(frame)-1])
		frame[len(frame)-1] = checksum[0]
	}

	//Make an entry in the forwarding table for the source address
	b.forwardingTable[string(sourceAddr)] = portNum

	//If an entry for the destination exists in the forwarding table then forward the frame there, else everywhere
	destPortNum, ok := b.forwardingTable[string(destAddr)]
	if ok {
		if b.isPartOfVlan(destPortNum, vlanId) {
			b.ports[destPortNum].GetAdapter().PutInBuffer(frame)
		}
	} else {
		for i, port := range b.ports {
			if i != portNum && b.isPartOfVlan(i, vlanId) {
				port.GetAdapter().PutInBuffer(frame)
			}
		}
	}
}

func (b *Bridge) isTrunk(portNum int) bool {
	return len(b.vlanTable[portNum]) > 2
}

func (b *Bridge) getVlanId(portNum int) uint16 {
	if len(b.vlanTable[portNum]) == 1 {
		return 0
	} else {
		return b.vlanTable[portNum][1]
	}
}

func (b *Bridge) isPartOfVlan(portNum int, vlanId uint16) bool {
	for _, i := range b.vlanTable[portNum] {
		if vlanId == i {
			return true
		}
	}

	return false
}
