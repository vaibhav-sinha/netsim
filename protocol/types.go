package protocol

import (
	"netsim/hardware"
)

type FrameConsumer interface {
	SendUp([]byte)
}

type FrameProducer interface {
	SendDown(data []byte, destAddr []byte, metadata []byte, sender Protocol)
}

type Protocol interface {
	FrameConsumer
	FrameProducer
	GetIdentifier() []byte
}

type L2Protocol interface {
	Protocol
	GetMTU() int
	GetAdapter() hardware.Adapter
}

type L3Protocol interface {
	Protocol
	SetL2Protocol(L2Protocol)
	GetL2Protocol() L2Protocol
}

type L4Protocol interface {
	Protocol
	SetL3Protocol(L3Protocol)
	GetL3Protocol() L3Protocol
}

type RouteProvider interface {
	GetNextHopForAddress([]byte) []byte
}

type AddressResolver interface {
	Resolve([]byte) []byte
}
