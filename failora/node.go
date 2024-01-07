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
	"os"
	"sync"
	"time"
)

/////////////////////////////////////////////
/////////      Data Structures      /////////
/////////////////////////////////////////////

type Peer struct {
	id             int32
	socketPath     string
	isTimeServer   bool
	latestElection int32
}

type SelfNode struct {
	mu                    sync.Mutex
	id                    int32
	me                    string
	leaderId              int32
	latestElection        int32
	peers                 map[int32]*Peer
	timeState             TimeServerData
	satisfactionWeights   map[int32]float64
	leaderInteractionData map[int32][]int32 // map[nodeA] = [interaction #1 time with nodeA, interaction #2 time with nodeA, interaction #3 time with nodeA...]
	isDead                bool              // this is for testing purposes so we can kill nodes for them to stop sending heartbeats
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

type HeartbeatArgs struct {
	/* whatever time interaction data needs to be transferred */
	latestElection int32
	leaderId       int32
}

type HeartbeatReply struct {
	/* whatever time interaction data needs to be transferred */
	latestElection int32
	leaderId       int32
	Err            Err
}

type Err string

// enum errors
const (
	OK             = "OK"
	ErrOldElection = "ErrOldElection"
	ErrNotLeader   = "ErrNotLeader"
)

const (
	compactnessWeight   = 0.5
	latencyWeight       = 0.5
	currentWeightFactor = 0.7
)

/////////////////////////////////////////////
/////////         Functions         /////////
/////////////////////////////////////////////

// NewSelfNode initializes this implementation of a node, and gives it the full peer list
func NewSelfNode(id int32, socketPath string, peers map[int32]*Peer) *SelfNode {
	node := &SelfNode{
		mu:                    sync.Mutex{},
		id:                    int32(id),
		me:                    socketPath,
		leaderId:              -1,
		latestElection:        -1,
		peers:                 peers,
		timeState:             TimeServerData{},
		satisfactionWeights:   make(map[int32]float64),
		leaderInteractionData: make(map[int32][]int32),
		isDead:                false,
	}

	go node.startRPCServer(socketPath)
	return node
}

// startRPCServer starts the RPC server for the node.
func (n *SelfNode) startRPCServer(socketPath string) {
	rpc.Register(n)
	os.Remove(socketPath) // Ensure the socket is not already in use
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatalf("Error starting RPC server for node %d: %v", n.id, err)
	}
	go rpc.Accept(ln)

	fmt.Printf("Node %d RPC server started on UNIX socket %s\n", n.id, socketPath)

	// start tick function
	for !n.isDead {
		time.Sleep(1 * time.Second)
		n.tick()
	}
}

// UpdateSatisfactionWeights updates the satisfaction weights based on interaction data.
// This function is intended to be called locally, not via RPC.
func (n *SelfNode) UpdateSatisfactionWeights(nodeID int, requestLatency float64) {
	// Function body as provided in your algorithm
	// ...
}

func (n *SelfNode) CalculateCompactnessScore() float64 {

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

// tick function which periodically pings the leader, and updates scores and interaction data
// in an implementation where nodes store state data, this would be a put/get request initiated by a client
func (n *SelfNode) tick() {
	n.mu.Lock()
	defer n.mu.Unlock()

	leader := n.peers[n.leaderId].socketPath

	args := &HeartbeatArgs{
		latestElection: n.latestElection,
		leaderId:       n.leaderId,
	}
	reply := &HeartbeatReply{}

	// ping leader
	Call(leader, "SelfNode.Heartbeat", args, &reply)

	// *update scores
	// *update interaction data

	// if the leader is down, start a new election
	if reply == nil {
		n.startElection()
	}

	// the reply tells us we are aware of an old election or leader,
	// we have to update our election data (leader ID and latest election)
	if reply.Err == ErrOldElection || reply.Err == ErrNotLeader {
		n.leaderId = reply.leaderId
		n.latestElection = reply.latestElection
	}

}

// start new election
func (n *SelfNode) startElection() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.latestElection = n.id

	return nil
}

/////////////////////////////////////////////
/////////        RPC Handlers       /////////
/////////////////////////////////////////////

// RPC Handler: ShareInteractionData allows a node to share its interaction data with this node.
func (n *SelfNode) ShareInteractionData(data InteractionData, reply *bool) error {
	n.UpdateSatisfactionWeights(data.NodeID, data.RequestLatency)
	*reply = true
	return nil
}

// RPC Handler: receive tick
func (n *SelfNode) Heartbeat(args *HeartbeatArgs, reply *HeartbeatReply) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	// if the peer that called this is on an old election, update them
	if args.latestElection < n.latestElection {
		reply.Err = ErrOldElection
		reply.latestElection = n.latestElection
		reply.leaderId = n.leaderId
		return nil
	}

	// if this node is not the leader, update them
	if n.leaderId != n.id {
		reply.Err = ErrNotLeader
		reply.latestElection = n.latestElection
		reply.leaderId = n.leaderId
		return nil
	}

	// if the peer that called this is on a newer election, update this node
	if args.latestElection > n.latestElection {
		n.latestElection = args.latestElection
		n.leaderId = args.leaderId
		reply.Err = OK
		return nil
	}

	// otherwise, all is good
	reply.Err = OK
	/* add any relevant time-interaction data to the reply, so the peer can update its scores */

	return nil
}

// RPC Handler: receive election
func (n *SelfNode) ReceiveElection() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	return nil
}
