package devices

import (
	"netsim/api"
	"netsim/hardware"
	"netsim/protocol"
	"netsim/protocol/l2"
	"netsim/protocol/l3"
	"netsim/protocol/l4"
)

type Computer struct {
	adapter         *hardware.EthernetAdapter
	l2Protocol      *l2.Ethernet
	ip              *l3.IP
	udp             *l4.UDP
	tcp             *l4.TCP
	routeProvider   *l3.StaticRouteProvider
	addressResolver *l3.StaticAddressResolver
}

func NewComputer(mac []byte, ipAddr []byte) *Computer {
	routeProvider := l3.NewStaticRouteProvider()
	addressResolver := l3.NewStaticAddressResolver()
	//Create the stack
	computer := &Computer{routeProvider: routeProvider, addressResolver: addressResolver}
	computer.adapter = hardware.NewEthernetAdapter(mac, false)
	ethernet := l2.NewEthernet(computer.adapter, nil)
	ip := l3.NewIP([][]byte{ipAddr}, false, nil, routeProvider, addressResolver)
	udp := l4.NewUDP()
	tcp := l4.NewTCP()

	//Set references
	ip.SetL2ProtocolForInterface(0, ethernet)

	//Arrange the stack
	ethernet.AddL3Protocol(ip)
	ip.AddL4Protocol(tcp)
	ip.AddL4Protocol(udp)
	tcp.AddL3Protocol(ip)

	computer.tcp = tcp
	computer.udp = udp
	return computer
}

func (c *Computer) AddAddress(ipAddr []byte, mac []byte) {
	c.addressResolver.Add(ipAddr, mac)
}

func (c *Computer) AddRoute(cidr *protocol.CIDR, gateway []byte) {
	c.routeProvider.Add(cidr, gateway, 0)
}

func (c *Computer) TurnOn() {
	c.adapter.TurnOn()
}

func (c *Computer) TurnOff() {
	c.adapter.TurnOff()
}

func (c *Computer) Run(runFunc func(computer *Computer)) {
	go runFunc(c)
}

func (c *Computer) NewSocket(domain int, channelType int, protocol int) *api.Socket {
	return api.NewSocket(c, domain, channelType, protocol)
}

func (c *Computer) GetAdapter() hardware.Adapter {
	return c.adapter
}

func (c *Computer) GetUDP() *l4.UDP {
	return c.udp
}

func (c *Computer) GetTCP() *l4.TCP {
	return c.tcp
}
