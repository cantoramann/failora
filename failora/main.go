package main

import "fmt"

func main() {

	// Example: Initialize nodes
	nodeA := NewSelfNode(1, "8080")
	nodeB := NewSelfNode(2, "8081")
	nodeC := NewSelfNode(3, "8082")
	nodeD := NewSelfNode(4, "8083")
	nodeE := NewSelfNode(5, "8084")
	nodeF := NewSelfNode(6, "8085")
	nodeG := NewSelfNode(7, "8086")
	nodeH := NewSelfNode(8, "8087")
	nodeI := NewSelfNode(9, "8088")
	nodeJ := NewSelfNode(10, "8089")
	nodeK := NewSelfNode(11, "8090")
	nodeL := NewSelfNode(12, "8091")
	nodeM := NewSelfNode(13, "8092")
	nodeN := NewSelfNode(14, "8093")
	nodeO := NewSelfNode(15, "8094")
	nodeP := NewSelfNode(16, "8095")
	nodeQ := NewSelfNode(17, "8096")

	// node A was the leader. It goes down.
	// node B discovered it first. B gets the timestamp of its discovery.
	// B broadcasts the discovery to all nodes.
	// Wait for 1 second. After, you are expected to have received (joins) from almost all other nodes * (address to the assumptions later)
	// Then B starts the voting. All nodes vote.

	// Satisfaction weights
	// response time
	// compactness

	// Additional nodes...

	nodeA.startRPCServer("8080")
	nodeB.startRPCServer("8081")

	// Main remains minimal - nodes operate autonomously after initialization
	fmt.Println("Nodes initialized and servers started", nodeA, nodeB)
	// The main function can block here or handle other master-level tasks
	select {} // Blocks forever
}
