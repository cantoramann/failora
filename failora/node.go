// package main

// import (
// 	"sync"
// 	"time"
// )

// const (
// 	compactnessWeight   = 0.5 // Adjust as needed
// 	latencyWeight       = 0.5 // Adjust as needed
// 	currentWeightFactor = 0.3
// )

// type NetworkNode struct {
// 	id           int32
// 	successor    *NetworkNode
// 	address      string
// 	isTimeServer bool
// 	mu           sync.Mutex
// }

// type SelfNode struct {
// 	NetworkNode
// 	timeState           TimeServerData
// 	satisfactionWeights map[int]float64
// }

// func NewSelfNode(id int32, address string) *SelfNode {
// 	node := &SelfNode{
// 		NetworkNode: NetworkNode{
// 			id:      id,
// 			address: address,
// 		},
// 		timeState: TimeServerData{
// 			current: CurrentTimeServerState{
// 				nodeId:           id,
// 				requestCount:     0,
// 				isTimeServer:     false,
// 				time:             0,
// 				lastFiveRequests: []int64{},
// 			},
// 		},
// 		satisfactionWeights: map[int]float64{},
// 	}

// 	return node
// }

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

// func (n *SelfNode) calculateCompactnessScore() float64 {

// 	// Add the current time to the list of last 5 requests
// 	if len(n.timeState.current.lastFiveRequests) >= 5 {
// 		n.timeState.current.lastFiveRequests = n.timeState.current.lastFiveRequests[1:] // Remove the oldest
// 	}
// 	n.timeState.current.lastFiveRequests = append(n.timeState.current.lastFiveRequests, time.Now().Unix())

// 	length := len(n.timeState.current.lastFiveRequests)
// 	if length < 2 {
// 		return 1.0 // Default high score if not enough data
// 	}

// 	var totalDifference int64
// 	for i := 1; i < length; i++ {
// 		difference := n.timeState.current.lastFiveRequests[i] - n.timeState.current.lastFiveRequests[i-1]
// 		// Smaller differences contribute more to the score
// 		totalDifference += difference
// 	}

// 	// Get the average difference
// 	averageDifference := totalDifference / int64(length-1)

// 	// Invert the average difference to get the score (smaller differences yield higher scores)
// 	compactnessScore := 1.0 / float64(averageDifference)
// 	return compactnessScore
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
)

type SelfNode struct {
	ID                  int
	isTimeServer        bool
	timeState           TimeState
	satisfactionWeights map[int]float64
	mu                  sync.Mutex
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
func NewSelfNode(id int, port string) *SelfNode {
	node := &SelfNode{
		ID:                  id,
		satisfactionWeights: make(map[int]float64),
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
		log.Fatalf("Error starting RPC server for node %d: %v", n.ID, err)
	}
	go rpc.Accept(ln)
	fmt.Printf("Node %d RPC server started on port %s\n", n.ID, port)
}

// RPC Method: ShareInteractionData allows a node to share its interaction data with this node.
func (n *SelfNode) ShareInteractionData(data InteractionData, reply *bool) error {
	n.UpdateSatisfactionWeights(data.NodeID, data.RequestLatency)
	*reply = true
	return nil
}

// Other necessary RPC methods can be defined here.
// ...
