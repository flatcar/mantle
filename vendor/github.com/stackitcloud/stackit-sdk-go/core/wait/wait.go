package wait

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
)

var RetryHttpErrorStatusCodes = []int{http.StatusBadGateway, http.StatusGatewayTimeout}

// AsyncActionCheck reports whether a specific async action has finished.
//   - waitFinished == true if the async action is finished, false otherwise.
//   - response contains data regarding the current state of the resource targeted by the async action, if applicable. If not applicable, T should be struct{}.
//   - err != nil if there was an error checking if the async action finished, or if it finished unsuccessfully.
type AsyncActionCheck[T any] func() (waitFinished bool, response *T, err error)

// AsyncActionHandler handles waiting for a specific async action to be finished.
type AsyncActionHandler[T any] struct {
	checkFn                  AsyncActionCheck[T]
	sleepBeforeWait          time.Duration
	throttle                 time.Duration
	timeout                  time.Duration
	tempErrRetryLimit        int
	IntermediateStateReached bool
}

// New initializes an AsyncActionHandler
func New[T any](f AsyncActionCheck[T]) *AsyncActionHandler[T] {
	return &AsyncActionHandler[T]{
		checkFn:           f,
		sleepBeforeWait:   0 * time.Second,
		throttle:          5 * time.Second,
		timeout:           30 * time.Minute,
		tempErrRetryLimit: 5,
	}
}

// SetThrottle sets the time interval between each check of the async action.
func (h *AsyncActionHandler[T]) SetThrottle(d time.Duration) *AsyncActionHandler[T] {
	h.throttle = d
	return h
}

// SetTimeout sets the duration for wait timeout.
func (h *AsyncActionHandler[T]) SetTimeout(d time.Duration) *AsyncActionHandler[T] {
	h.timeout = d
	return h
}

// SetSleepBeforeWait sets the duration for sleep before wait.
func (h *AsyncActionHandler[T]) SetSleepBeforeWait(d time.Duration) *AsyncActionHandler[T] {
	h.sleepBeforeWait = d
	return h
}

// SetTempErrRetryLimit sets the retry limit if a temporary error is found.
// The list of temporary errors is defined in the RetryHttpErrorStatusCodes variable.
func (h *AsyncActionHandler[T]) SetTempErrRetryLimit(l int) *AsyncActionHandler[T] {
	h.tempErrRetryLimit = l
	return h
}

// WaitWithContext starts the wait until there's an error or wait is done
func (h *AsyncActionHandler[T]) WaitWithContext(ctx context.Context) (res *T, err error) {
	if h.throttle == 0 {
		return nil, fmt.Errorf("throttle can't be 0")
	}

	ctx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	// Wait some seconds for the API to process the request
	time.Sleep(h.sleepBeforeWait)

	ticker := time.NewTicker(h.throttle)
	defer ticker.Stop()

	var retryTempErrorCounter = 0
	for {
		done, res, err := h.checkFn()
		if err != nil {
			retryTempErrorCounter, err = h.handleError(retryTempErrorCounter, err)
			if err != nil {
				return res, err
			}
		}
		if done {
			return res, nil
		}

		select {
		case <-ctx.Done():
			return res, fmt.Errorf("WaitWithContext() has timed out")
		case <-ticker.C:
			continue
		}
	}
}

func (h *AsyncActionHandler[T]) handleError(retryTempErrorCounter int, err error) (int, error) {
	oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
	if !ok {
		return retryTempErrorCounter, fmt.Errorf("found non-GenericOpenApiError: %w", err)
	}
	// Some APIs may return temporary errors and the request should be retried
	if !utils.Contains(RetryHttpErrorStatusCodes, oapiErr.StatusCode) {
		return retryTempErrorCounter, err
	}
	retryTempErrorCounter++
	if retryTempErrorCounter == h.tempErrRetryLimit {
		return retryTempErrorCounter, fmt.Errorf("temporary error was found and the retry limit was reached: %w", err)
	}
	return retryTempErrorCounter, nil
}
