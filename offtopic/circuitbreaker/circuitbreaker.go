package circuitbreaker

import (
	"context"
	"time"
)

type xRequestBreaker struct {
	maxAvgReqTime    time.Duration      //When average duration grows above this level, circuit flips
	off              bool               //could slap behind a mutex or atomic but idgaf
	recoveryAverage  time.Duration      //After flipping the Circuit, flip back on after this point
	xRequestsToTrack int                //How many requests factor into the average
	reqCompleteSig   chan time.Duration // communication layer between goroutines that signals completion of a request
}

func (x *xRequestBreaker) FlippedOff() bool {
	return x.off
}

//Updates the state of the circuit- whether or not it's been flipped
func (x *xRequestBreaker) calculateAverage(durs []time.Duration) {
	var tot time.Duration
	for i := 0; i < x.xRequestsToTrack; i++ {
		tot += durs[i]
	}

	avg := tot / time.Duration(x.xRequestsToTrack)

	if avg > x.maxAvgReqTime {
		x.off = true
		return
	}

	if x.off && avg < x.recoveryAverage {
		x.off = false
		return
	}
}

func (x *xRequestBreaker) launchRunner(ctx context.Context) {
	// allocate space for the actual pool of requests to use
	requestTimes := make([]time.Duration, x.xRequestsToTrack)

	// only update the average every 1/5 second (arbitrarily chosen number)
	//TODO: choose better sampling mechanism
	updateEvery := time.NewTicker(200 * time.Millisecond)

	currentInsertIndex := 0
	numReceived := 0

	for {
		select {
		case <-updateEvery.C: // calculate the average when the ticker fires
			if numReceived < x.xRequestsToTrack {
				continue
			}
			x.calculateAverage(requestTimes)
		case d := <-x.reqCompleteSig: //a request has completed, add it to the pool
			numReceived++
			//TODO: instead of blindly inserting all requests into the pool, use a sample-rate
			//because http requests can happen at rates of >= 10000 per second, meaning we shouldn't
			//be updating on all requests
			requestTimes[currentInsertIndex] = d
			currentInsertIndex++
			if currentInsertIndex >= x.xRequestsToTrack {
				currentInsertIndex = 0
			}
		case <-ctx.Done():
			return
		}
	}
}

//NewBreaker creates an instance of a circuit breaker.
// - Immediately launches the polling goroutine in
// - ctx allows external processes to close out the Runner via context.WithCancel. Once it
//has been canceled, the breaker must be discarded and a new one created by calling NewBreaker
//
//http servers should be built to handle large volumes of requests. It is recommended
//to make a new breaker for different HTTP methods/operations, then to use custom logic to choose
//whether or not to signal
func NewBreaker(ctx context.Context, flipAtAvgOf, recoverAtAvgOf time.Duration, ) *xRequestBreaker {
	breaker := &xRequestBreaker{
		maxAvgReqTime:    flipAtAvgOf,
		recoveryAverage:  recoverAtAvgOf,
		xRequestsToTrack: 100, //hardcoding to 100 for now
		reqCompleteSig:   make(chan time.Duration),
	}

	breaker.launchRunner(ctx)

	return breaker
}
