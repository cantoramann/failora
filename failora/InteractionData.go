package main

type CurrentInteractionState struct {
	nodeId       uint32
	requestCount uint64
	totalLatency uint64
	startEpoch   uint64
}
