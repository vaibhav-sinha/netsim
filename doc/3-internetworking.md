## Inter-networking

Up until now, the problem was to connect nodes directly with each other and provide networking capabilities to that
small network. It is not possible to create a global scale network by interconnecting all nodes directly. A single 
Ethernet segment can have maximum 1024 nodes. A point-to-point link connects only 2 nodes. Hence, we need a way to
connect multiple small networks to create a large one.

Also, not all networks may be based on same link layer technology. We also need a way to link heterogeneous networks.
Devices which connect multiple links together to create larger network are called switches or bridges. On the other hand,
devices which connect multiple heterogeneous networks are called routers or gateways. Once multiple links are connected to
create larger networks, and subsequently those networks are connected to create even larger networks, then there end up
being multiple routes from one node to another. The problem of figuring out the best route between nodes is called routing.

### Switching
Switching is a mechanism to connect links to form larger networks. Multiple switches can also be connected to each other
which can make the network even larger. A switch receives frames from one network and forwards them to other networks
connected to it. To decide which port a frame has to be sent out, the switch looks at some bytes containing an identifier.
There are two approaches to sending the frames - data (connectionless) and virtual-circuit (connection-oriented).

#### Datagram
In case of datagrams, each frame contains the full destination address. The switch maintains a forwarding table which
maps addresses to ports. This table is learnt dynamically. When the switch receives a frame with a destination address
for which it does not have any entry in its forwarding table, then it sends it out all the ports. 

1. A packet can be sent anytime to any destination
2. The source does not know if a packet can be delivered to destination or not
3. Each packet is sent independently of others
4. A link or switch failure in the network might not cause any disruption

#### Virtual Circuit
This is a two-stage approach. First a connection is created from source to destination and then packets are sent on that
connection. For making a connection, each switch in the network that is used in the connection maintains a connection
state in a VC table. The network state contains a source identifier along with the source port, and whenever it receives a packet
on that port with that identifier, it sends it out destination port after changing the identifier to destination identifier.

The connection may be created by an administrator or by sending packets in the connection phase (called signalling)

### Limitations
1. Does not scale because of multicast
2. Cannot be used to connect heterogeneous networks which use different addressing schemes
3. The addressing is not hierarchical hence the need to store forwarding information for all nodes, which is not feasible in internet scale networks

### Virtual LAN
VLAN is a technology to increase the scalability of switched networks by dividing it into multiple virtual networks.
Each VLAN is represented by a number, and a packet can travel from one segment to another if they both belong to the same VLAN. This
reduces the number of hosts which will receive any broadcast packet.