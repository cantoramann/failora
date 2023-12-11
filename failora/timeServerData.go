package main

type CurrentTimeServerState struct {
	nodeId           int32
	requestCount     int64
	isTimeServer     bool
	time             int64
	lastFiveRequests []int64
}

type TimeServerData struct {
	current CurrentTimeServerState
}
