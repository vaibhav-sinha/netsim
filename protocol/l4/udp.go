package l4

import (
	"netsim/protocol"
	"netsim/protocol/l3"
)

/*
We will implement UDP which is a simple de-multiplexing protocol for process-to-process communication. I will keep the
implementation as simple as possible. That means no two processes can listen on same port no matter what. This is very
different than what happens in real OS implementations, if host has multiple IP addresses or SO_REUSEADDR is set. Also,
since UDP only needs to run on hosts, I am making the assumption that there is just one network interface on the host
and hence do not need to deal multi-host scenario.

Packet Format

SrcPort		- 2 byte
DestPort	- 2 byte
Length		- 2 byte
Checksum	- 1 byte
Data		- No fixed length
*/
type UDP struct {
	identifier   []byte
	intfs        []*l3.IP
	portBindings map[int64]*binding
}

/*
Next 3 methods make this a Protocol
*/
func (u *UDP) GetIdentifier() []byte {
	return u.identifier
}

func (u *UDP) SendUp(data []byte, metadata []byte, sender protocol.Protocol) {
	//Not used since UDP works on pull model rather than push
}

func (u *UDP) SendDown(data []byte, destAddr []byte, metadata []byte, sender protocol.Protocol) {

}

func (u *UDP) SetL3Protocol(l3Protocol protocol.L3Protocol) {
}

/*
Internal struct to track bindings
*/
type binding struct {
	ip   []byte
	port int16
}
