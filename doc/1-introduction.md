## High level picture
1. Computer networks provide a general infrastructure over which a lot of different applications can be built.
2. At the lowest level, a network may consist of two computers connected by a link (point-to-point) or multiple computers connected over a link (multiple-access)
3. All nodes directly connected over the same link is not scalable. Hence, we rely on indirect connectivity.
4. A node connected to more than one node would forward data from one node to other nodes. Such networks are called switched networks.
5. Computer networks are mostly packet-switched where discrete blocks of data are sent between nodes.
6. Switches in packet-switched networks first store the entire packet and then send them.
7. Similar to multiple computers being connected to each other, it is possible to connect multiple networks with each other.
8. Nodes which are connected to two or more networks are called gateways or routers.
9. Once we have connectivity, then to be able to send a message from node to another (connected directly or indirectly), we need some sort of addressing, using which the switches and routers can decide where to send the data.
10. Since multiple nodes might want to communicate simultaneously over shared links, we need a way to multiplex data on the link. To do this we can do time multiplexing (fixed time slots), frequency multiplexing (fixed frequency slots) or statistical multiplexing (fixed packet size).

## Network properties
1. Data Rate (Bandwidth)
2. Latency
3. RTT
4. Delay Bandwidth Product

## Abstraction
1. Applications want a simplified view of network where they can assume the network to be a pipe that connects processes
2. Different applications have different requirements from the network (req-reply, file-sharing, video-streaming)
3. Networking code is hence implemented in layers so that lower layers provide certain services to higher layers and multiple different higher layers can be implemented to cater to different requirements
