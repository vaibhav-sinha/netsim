package protocol

import "netsim/utils"

var DefaultRouteCidr = &CIDR{
	Address: []byte{0, 0, 0, 0},
	Mask:    0,
}

var (
	IP  = utils.HexStringToBytes("0800")
	UDP = utils.HexStringToBytes("11")
	TCP = utils.HexStringToBytes("06")
)
