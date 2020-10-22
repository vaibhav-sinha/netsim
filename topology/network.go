package topology

import "netsim/devices"

type Network struct {
	members map[string]interface{}
}

func NewNetwork(config map[string]interface{}) *Network {
	n := &Network{
		members: map[string]interface{}{},
	}
	n.createFromConfig(config)
	return n
}

func (n *Network) createFromConfig(config map[string]interface{}) {

}

func (n *Network) GetComputer(name string) *devices.Computer {
	c, ok := n.members[name]
	if ok {
		return c.(*devices.Computer)
	}

	return nil
}
