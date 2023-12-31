// package main

// import (
// 	"sync"
// 	"time"
// )

// func (n *SelfNode) UpdateSatisfactionWeights(nodeID int, requestLatency float64) {
// 	if n.timeState.current.isTimeServer || requestLatency <= 0 {
// 		return
// 	}

// 	// Arbitrary threshold
// 	if requestLatency < 0.001 {
// 		requestLatency = 0.001
// 	}

// 	// Get the latency score
// 	latencyScore := 1.0 / requestLatency

// 	// Get the compactness score
// 	compactnessScore := n.calculateCompactnessScore()
// 	combinedScore := (compactnessWeight * compactnessScore) + (latencyWeight * latencyScore)

// 	previousWeight, exists := n.satisfactionWeights[nodeID]
// 	if !exists {
// 		previousWeight = combinedScore
// 	}

// 	averageWeight := currentWeightFactor*combinedScore + (1-currentWeightFactor)*previousWeight
// 	n.satisfactionWeights[nodeID] = averageWeight
// }

// func (n *SelfNode) UpdateTimeServer(newTimeServerId int32) {
// 	n.timeState.current.requestCount = 0
// 	n.timeState.current.nodeId = newTimeServerId

// 	if newTimeServerId == n.id {
// 		n.timeState.current.isTimeServer = true
// 	}
// }

package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"sync"
	"time"
)

type Peer struct {
	id           int32
	address      string
	isTimeServer bool
	mu           sync.Mutex
	election     int32
}

type SelfNode struct {
	id                    int32
	address               string
	leaderId              int32
	mu                    sync.Mutex
	peers                 map[int32]*Peer
	timeState             TimeServerData
	satisfactionWeights   map[int32]float64
	leaderInteractionData map[int32][]int32 // map[nodeA] = [interaction #1 time with nodeA, interaction #2 time with nodeA, interaction #3 time with nodeA...]
	election              int32
}

type TimeState struct {
	current CurrentTimeState
}

type CurrentTimeState struct {
	isTimeServer bool
}

type InteractionData struct {
	NodeID         int
	RequestLatency float64
}

const (
	compactnessWeight   = 0.5
	latencyWeight       = 0.5
	currentWeightFactor = 0.7
)

// NewSelfNode creates a new SelfNode with Given ID and initializes its RPC server.
func NewSelfNode(id int, port string, peers []Peer) *SelfNode {
	node := &SelfNode{
		id:                  int32(id),
		satisfactionWeights: make(map[int32]float64),
		peers:               make(map[int32]*Peer),
	}

	go node.startRPCServer(port)
	return node
}

// UpdateSatisfactionWeights updates the satisfaction weights based on interaction data.
// This function is intended to be called locally, not via RPC.
func (n *SelfNode) UpdateSatisfactionWeights(nodeID int, requestLatency float64) {
	// Function body as provided in your algorithm
	// ...
}

// calculateCompactnessScore is a placeholder for your actual implementation.
func (n *SelfNode) calculateCompactnessScore() float64 {
	// Dummy implementation
	return 1.0
}

// startRPCServer starts the RPC server for the node.
func (n *SelfNode) startRPCServer(port string) {
	rpc.Register(n)
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Error starting RPC server for node %d: %v", n.id, err)
	}
	go rpc.Accept(ln)
	fmt.Printf("Node %d RPC server started on port %s\n", n.id, port)
}

// RPC Method: ShareInteractionData allows a node to share its interaction data with this node.
func (n *SelfNode) ShareInteractionData(data InteractionData, reply *bool) error {
	n.UpdateSatisfactionWeights(data.NodeID, data.RequestLatency)
	*reply = true
	return nil
}

// Other necessary RPC methods can be defined here.
// ...

func (n *SelfNode) calculateCompactnessScore() float64 {

	// Add the current time to the list of last 5 requests
	if len(n.timeState.current.lastFiveRequests) >= 5 {
		n.timeState.current.lastFiveRequests = n.timeState.current.lastFiveRequests[1:] // Remove the oldest
	}
	n.timeState.current.lastFiveRequests = append(n.timeState.current.lastFiveRequests, time.Now().Unix())

	length := len(n.timeState.current.lastFiveRequests)
	if length < 2 {
		return 1.0 // Default high score if not enough data
	}

	var totalDifference int64
	for i := 1; i < length; i++ {
		difference := n.timeState.current.lastFiveRequests[i] - n.timeState.current.lastFiveRequests[i-1]
		// Smaller differences contribute more to the score
		totalDifference += difference
	}

	// Get the average difference
	averageDifference := totalDifference / int64(length-1)

	// Invert the average difference to get the score (smaller differences yield higher scores)
	compactnessScore := 1.0 / float64(averageDifference)
	return compactnessScore
}
