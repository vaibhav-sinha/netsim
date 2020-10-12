package api

import "netsim/protocol/l4"

type Host interface {
	GetUDP() *l4.UDP
}
