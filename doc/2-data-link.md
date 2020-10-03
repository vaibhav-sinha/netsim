## Data Link Layer
Once nodes are connected by a link, there are 5 things needed so that they can communicate

1. Encoding
2. Framing
3. Error Detection
4. Reliability
5. Media Access Control

There are 3 kinds of link scenario which are interesting

1. Point-to-point
2. Carrier Sense Multiple Access (for example, Ethernet)
3. Wireless (for example, WiFi)

### Encoding
Encoding requires that bits are somehow encoded as electromagnetic waves that travel on the link. For a wired connection we can assume that high voltage represents bit 1 and low voltage represents 0. In reality though, there is more that is needed to be done to do clock recovery and to prevent baseline wander.

### Framing
Since we are concerned with packet-switched networks, the lowest unit for data for us is a packet, not a bit. Hence we need a way to differentiate where a frame starts and where it ends.
There are 3 major ways in which framing can be done:

1. Sentinel based approaches - Special bit/byte sequence denotes start and end of frames. Point-to-point protocol is based on this. PPP frame format has fields whose length can vary and is decided through negotiation using the Link Control Protocol.
2. Byte-counting approaches - The frame contains a length field that denotes the length of the frame
3. Clock based approach - A bit like sentinel based approach but more complex. Used in long distance optical networks. No character/bit stuffing needed.

### Error Detection
Since data corruption might happen on the link, we need a way to detect errors, and if possible also correct them so that there is no need to drop the frame. Error detection and correction depends on what algorithm is used and how much redundant data is sent.
Few ways to encode error detection information in the frame are:

1. Parity - 1D, 2D
2. Checksum
3. Cyclic Redundancy Check

### Reliability
Since frames can get corrupted and hence dropped, there needs to be a way to recover them to give a view to reliable link to higher level protocol. To do this, a retransmission logic would need to be implemented in the protocol. Besides, providing all the frames, it might also be a requirement to provide all the frames in order. While these can be implemented at Data Link Layer, most technologies don't do that. Instead, these are implemented at Transport Layer or Application Layer.
To implement redelivery, acknowledgements and timeouts are used. There are 3 mechanisms to implement redelivery:

1. Stop-and-Wait - Ordering and reliability. Bad performance.
2. Sliding Window - Ordering and reliability. Complex.
3. Concurrent Logical Channels - Reliability but no ordering.

### Media Access Control
If multiple nodes are connected to the same link, there has to be a way to determine who is allowed to send a frame on the link at a given time so that collisions don't happen. Different technologies use different mechanisms to give access to link to nodes depending upon the contraints imposed by the physical link.

1. Ethernet (CSMA/CD) - Nodes sense the link to find if it is free and then send a frame. If they detect a collision, they backoff and try again after some time.
2. Token Ring - Nodes are arranged in a ring. There is a token which is passed from one node to another. The node that has the token gets to send frames.
3. Wireless (CSMA/CA) - Node sends a small Ready-to-Send packet to the destination node first. If the destination node hears it, it sends Clear-to-Send and only then a frame is sent by the source.