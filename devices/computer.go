package devices

import (
	"netsim/hardware"
	"netsim/protocol"
	"netsim/protocol/l2"
)

type Computer struct {
	adapter     *hardware.EthernetAdapter
	l2Protocol  *l2.SimpleEthernet
	l3Protocols []protocol.Protocol
}
