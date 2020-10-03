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
	GetAdapter() hardware.Adapter
}

type L3Protocol interface {
	Protocol
	SetL2Protocol(L2Protocol)
}
