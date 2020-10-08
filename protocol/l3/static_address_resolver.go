package l3

/*
Static Address Resolution
*/
type StaticAddressResolver struct {
	macTable map[int64][]byte
}

func NewStaticAddressResolver() *StaticAddressResolver {
	return &StaticAddressResolver{macTable: map[int64][]byte{}}
}

func (s *StaticAddressResolver) Add(ipAddr []byte, mac []byte) {
	key := int64(-1)
	if ipAddr != nil {
		key = s.ipToKey(ipAddr)
	}
	s.macTable[key] = mac
}

func (s *StaticAddressResolver) Resolve(ipAddr []byte) []byte {
	return s.macTable[s.ipToKey(ipAddr)]
}

func (s *StaticAddressResolver) ipToKey(ipAddr []byte) int64 {
	return int64(ipAddr[0]) + 10 ^ 3*int64(ipAddr[1]) + 10 ^ 6*int64(ipAddr[2]) + 10 ^ 9*int64(ipAddr[3])
}
