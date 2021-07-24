package raft

import (
	"fmt"
	"math/rand"
	"time"
)

// NodeState indicates the state of current node.
type NodeState string

const (
	Follower  = "follower"
	Candidate = "candidate"
	Leader    = "leader"
)

// debugEnabled tells if debugging enabled
const debugEnabled = false

func Debug(format string, a ...interface{}) (n int, err error) {
	if debugEnabled {
		n, err = fmt.Printf(format, a...)
	}
	return
}

// RandomDuration generates a random duration based on upper and lower timestamp.
func RandomDuration(lower, upper time.Duration) time.Duration {
	num := rand.Int63n(upper.Nanoseconds()-lower.Nanoseconds()) + lower.Nanoseconds()
	return time.Duration(num) * time.Nanosecond
}

// ResetTimer resets the timer with the duration.
func ResetTimer(timer *time.Timer, d time.Duration) {
	if !timer.Stop() {
		select {
		case <-timer.C: // try to drain from the channel
		default:
		}
	}
	timer.Reset(d)
}
