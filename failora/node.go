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

type Peer struct {
	id             uint32
	socketPath     string
	isTimeServer   bool
	latestElection uint32
}

type SelfNode struct {
	mu                      sync.Mutex
	id                      uint32
	me                      string
	leaderId                uint32
	latestElection          uint32
	electionState           Election // necessary state of the current OR most recently decided election (whichever is more recent)
	peers                   map[uint32]*Peer
	currentInteractionState CurrentInteractionState
	previousInteractionData map[uint32]struct {
		score float64
		count uint32
	}
	isDead     bool // this is for testing purposes so we can kill nodes for them to stop sending heartbeats
	startEpoch uint64
}

// necessary state of the current OR most recently decided election (whichever is more recent)
// if there is no current election in progress, this will be the most recent election, latestElection
// if there is an election in progress, this will be latestElection + 1
type Election struct {
	electionLeaderId uint32 //not to be confused with the system leader, the election leader is the node that initiated the election
	electionId       uint32
	votes            map[uint32]uint32 // map[nodeA] = nodeB, where nodeA voted for nodeB
	electionComplete bool
	newLeaderId      uint32 // the new leader of the system, once the election is complete
}

type TimeState struct {
	current CurrentTimeState
}

type CurrentTimeState struct {
	isTimeServer bool
}

type InteractionData struct {
	NodeID         uint32
	RequestLatency float64
}

type HeartbeatArgs struct {
	/* whatever time interaction data needs to be transferred */
	latestElection uint32
	leaderId       uint32
}

type HeartbeatReply struct {
	/* whatever time interaction data needs to be transferred */
	latestElection uint32
	leaderId       uint32
	Err            Err
}

type NewElectionArgs struct {
	newElectionId     uint32
	oldLeaderId       uint32
	electionInitiated bool // if this is true, then the electionLeader already received OK from all peers re. starting a new election
	electionLeaderId  uint32
}

type NewElectionReply struct {
	currentElectionId uint32 // will be used if the responding node knows of a newer completed election (disagree to start new election)
	currentLeaderId   uint32 // "
	Err               Err    // will reply OK if the responding node agrees to start a new election on newElectionId
	vote              uint32 // only if the election is already initiated and we are casting our vote to the electionLeader
}

type ReceiveCompletedElectionArgs struct {
	electionState Election
}

type ReceiveCompletedElectionReply struct {
	Err Err
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

// NewSelfNode initializes this implementation of a node, and gives it the full peer list
func NewSelfNode(id uint32, socketPath string, peers map[uint32]*Peer) *SelfNode {
	node := &SelfNode{
		mu:                      sync.Mutex{},
		id:                      uint32(id),
		me:                      socketPath,
		leaderId:                0,
		latestElection:          0,
		electionState:           Election{},
		peers:                   peers,
		currentInteractionState: CurrentInteractionState{},
		previousInteractionData: make(map[uint32]struct {
			score float64
			count uint32
		}),
		isDead:     false,
		startEpoch: getCurrentTime(),
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

func (n *SelfNode) CalculateNewSatisfactionScore(prevLeaderId uint32) {
	currentTime := getCurrentTime()
	leadershipDuration := currentTime - n.currentInteractionState.startEpoch

	normalizedDuration := float64(leadershipDuration) / float64(currentTime-n.startEpoch)
	normalizedRequestCount := float64(n.currentInteractionState.requestCount) / float64(n.currentInteractionState.totalLatency)
	newScore := normalizedDuration * normalizedRequestCount

	leadershipCount := n.previousInteractionData[prevLeaderId].count + 1
	prevLeadershipScore := n.previousInteractionData[prevLeaderId].score

	if leadershipCount == 1 {
		n.previousInteractionData[prevLeaderId] = struct {
			score float64
			count uint32
		}{newScore, leadershipCount}
	} else {
		overallScore := ((0.8 * newScore) * (0.2 * prevLeadershipScore)) / float64(leadershipCount)
		n.previousInteractionData[prevLeaderId] = struct {
			score float64
			count uint32
		}{overallScore, leadershipCount}
	}
}

func (n *SelfNode) UpdateLeaderInteractionData() {
	n.currentInteractionState.totalLatency += getCurrentTime()
	n.currentInteractionState.requestCount++
}

// tick function which periodically pings the leader, and updates scores and interaction data
// in an implementation where nodes store state data, this would be a put/get request initiated by a client rather than a periodic ping
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

	// update response metadata
	n.UpdateLeaderInteractionData()
	// *update interaction data

	// if the leader is down and there is no current ONGOING election, start a new election
	if reply == nil {
		elecDone := false
		elecDone = n.startElection()
		// continue waiting in a loop while startElection() hasn't returned yet
		for !elecDone {
			time.Sleep(1 * time.Second)
		}
	}

	// the reply tells us we are aware of an old election or leader,
	// we have to update our election data (leader ID and latest election)
	if reply.Err == ErrOldElection || reply.Err == ErrNotLeader {
		n.leaderId = reply.leaderId
		n.latestElection = reply.latestElection
	}

}

// start new election
func (n *SelfNode) startElection() bool {
	n.mu.Lock()
	defer n.mu.Unlock()

	// broadcast new election number latestElection + 1 AND the latest election data to all peers
	// the first gossip is an attempt to initiate a new election
	firstGossipArgs := &NewElectionArgs{
		newElectionId:     n.latestElection + 1,
		oldLeaderId:       n.leaderId,
		electionInitiated: false,
		electionLeaderId:  n.id,
	}

	// keep track of the number of OKs we receive
	oks := 0
	// votes
	votes := make(map[uint32]uint32)

	// call NewElection RPC handler on all peers
	for _, peer := range n.peers {
		reply := &NewElectionReply{}

		Call(peer.socketPath, "SelfNode.NewElection", firstGossipArgs, &reply)

		if reply.Err == OK {
			oks++
			// add the vote to the votes map
			votes[peer.id] = reply.vote
		} else if reply.Err == ErrOldElection {
			// if we receive an ErrOldElection, update our latestElection and leaderId
			n.latestElection = reply.currentElectionId
			n.leaderId = reply.currentLeaderId
			// end the initiation attempt by returning true to the tick() function (which will run again with the NEW leaderId saved)
			return true
		}
	}

	// if we received OK from a majority of peers, without being told by any peer that we had an old election,
	// we can declare the election done, and tally up the votes to select a new leader
	// then we can append the new leader to the election state, set electionComplete to true, then broadcast election state to all peers
	if oks > len(n.peers)/2 {
		// update the local election state
		n.electionState.electionId = n.latestElection + 1
		n.electionState.electionLeaderId = n.id
		n.electionState.electionComplete = true
		n.electionState.votes = votes
		n.electionState.votes[n.id] = n.castVote()

		voteCounts := make(map[uint32]uint32)
		maxVotes := uint32(0)
		newLeader := uint32(0)
		for _, vote := range votes {
			voteCounts[vote]++
			if voteCounts[vote] > maxVotes {
				maxVotes = voteCounts[vote]
				newLeader = vote
			}
		}

		n.electionState.newLeaderId = newLeader

		// broadcast the election state to all peers
		oks := 0
		for oks <= len(n.peers)/2 {
			oks = 0
			for _, peer := range n.peers {
				reply := &ReceiveCompletedElectionReply{}
				args := &ReceiveCompletedElectionArgs{
					electionState: n.electionState,
				}

				Call(peer.socketPath, "SelfNode.ReceiveCompletedElection", args, &reply)

				if reply.Err == OK {
					oks++
				}
			} // END FOR EACH PEER
		} // END WHILE OKS < MAJORITY

	} // END IF

	// if the OKs were not from a majority of peers, returning true below to the tick() function will cause it to run again with the OLD leaderId
	return true
} // END startElection()

func (n *SelfNode) castVote() uint32 {
	// get the maximum score from the previousInteractionData map as well as the key associated with it
	maxScore := float64(0)
	maxScoreKey := uint32(0)
	for key, value := range n.previousInteractionData {
		if value.score > maxScore {
			maxScoreKey = key
		}
	}

	return uint32(maxScoreKey)
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
func (n *SelfNode) NewElection(args *NewElectionArgs, reply *NewElectionReply) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	// if the peer that called this is on an old election, update them
	if args.newElectionId <= n.latestElection {
		reply.Err = ErrOldElection
		reply.currentElectionId = n.latestElection
		reply.currentLeaderId = n.leaderId
		return nil
	}

	// otherwise, just cast a vote for the election leader and return OK
	reply.Err = OK
	reply.vote = n.castVote()

	return nil
}

// RPC Handler: receive election state
func (n *SelfNode) ReceiveCompletedElection(args *ReceiveCompletedElectionArgs, reply *ReceiveCompletedElectionReply) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	// coerce this node to update the local election state and all local data
	n.electionState = args.electionState
	n.latestElection = args.electionState.electionId
	n.leaderId = args.electionState.newLeaderId

	reply.Err = OK
	return nil
}

// Returns the current epoch time in nanoseconds
func getCurrentTime() uint64 {
	return uint64(time.Now().UTC().UnixNano())
}
