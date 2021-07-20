package common

import (
	"fmt"
	"math/rand"
	"time"
)

// debugEnabled tells if debugging enabled
const debugEnabled = false

func Debug(format string, a ...interface{}) (n int, err error) {
	if debugEnabled {
		n, err = fmt.Printf(format, a...)
	}
	return
}

func RandomDuration(lower, upper time.Duration) time.Duration {
	num := rand.Int63n(upper.Nanoseconds()-lower.Nanoseconds()) + lower.Nanoseconds()
	return time.Duration(num)*time.Nanosecond
}

func ResetTimer(timer *time.Timer, d time.Duration) {
	if !timer.Stop() {
		select {
		case <-timer.C: // try to drain from the channel
		default:
		}
	}
	timer.Reset(d)
}
