package main

import (
	"fmt"
	"os"
	"strconv"
)

func port(tag string, host int) string {
	s := "/var/tmp/824-"
	s += strconv.Itoa(os.Getuid()) + "/"
	os.Mkdir(s, 0777)
	s += "failora-"
	s += strconv.Itoa(os.Getpid()) + "-"
	s += tag + "-"
	s += strconv.Itoa(host)
	return s
}

func main() {

	tag := "failora"

	// make 5 Peers with socket paths using port() function
	peer1 := Peer{
		id:             1,
		socketPath:     port(tag, 1),
		isTimeServer:   false,
		latestElection: -1,
	}
	peer2 := Peer{
		id:             2,
		socketPath:     port(tag, 2),
		isTimeServer:   false,
		latestElection: -1,
	}
	peer3 := Peer{
		id:             3,
		socketPath:     port(tag, 3),
		isTimeServer:   false,
		latestElection: -1,
	}
	peer4 := Peer{
		id:             4,
		socketPath:     port(tag, 4),
		isTimeServer:   false,
		latestElection: -1,
	}
	peer5 := Peer{
		id:             5,
		socketPath:     port(tag, 5),
		isTimeServer:   false,
		latestElection: -1,
	}

	// make a map of the peer ids to the peers
	peers := make(map[int32]*Peer)
	peers[peer1.id] = &peer1
	peers[peer2.id] = &peer2
	peers[peer3.id] = &peer3
	peers[peer4.id] = &peer4
	peers[peer5.id] = &peer5

	// NewSelfNode() initializes the node state and starts the RPC server
	node1 := NewSelfNode(peer1.id, peer1.socketPath, peers)
	node2 := NewSelfNode(peer2.id, peer2.socketPath, peers)
	node3 := NewSelfNode(peer3.id, peer3.socketPath, peers)
	node4 := NewSelfNode(peer4.id, peer4.socketPath, peers)
	node5 := NewSelfNode(peer5.id, peer5.socketPath, peers)

	// node A was the leader. It goes down.
	// node B discovered it first. B gets the timestamp of its discovery.
	// B broadcasts the discovery to all nodes.
	// Wait for 1 second. After, you are expected to have received (joins) from almost all other nodes * (address to the assumptions later)
	// Then B starts the voting. All nodes vote.

	// Satisfaction weights
	// response time
	// compactness

	// Additional nodes...

	// Main remains minimal - nodes operate autonomously after initialization
	fmt.Println("Nodes initialized and servers started", node1, node2, node3, node4, node5)
	// The main function can block here or handle other master-level tasks
	select {} // Blocks forever
}
