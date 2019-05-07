package mem

import (
	"context"
	"sync"
	"time"

	"github.com/oasislabs/developer-gateway/mqueue/core"
)

const (
	maxElementsPerQueue  = 1024
	maxInactivityTimeout = 10 * time.Minute
)

type insertRequest struct {
	Element core.Element
	Out     chan<- error
}

type retrieveRequest struct {
	Offset uint64
	Count  uint
	Out    chan<- core.Elements
}

type discardRequest struct {
	Offset uint64
	Out    chan<- error
}

type nextRequest struct {
	Out chan<- uint64
}

// Worker implements a very simple messaging queue-like
// functionality serving requests for a single queue.
type Worker struct {
	lastProcessedRequest uint64
	key                  string
	doneCh               chan<- string
	inCh                 chan interface{}
	wg                   sync.WaitGroup
	queue                *OrderedQueue
}

// NewWorker creates a new instance of a worker
func NewWorker(ctx context.Context, key string, doneCh chan<- string) *Worker {
	w := &Worker{
		lastProcessedRequest: uint64(time.Now().Unix()),
		key:                  key,
		doneCh:               doneCh,
		inCh:                 make(chan interface{}),
		wg:                   sync.WaitGroup{},
		queue:                NewOrderedQueue(maxElementsPerQueue),
	}

	w.startLoop(ctx)
	return w
}

func (w *Worker) Stop() {
	w.wg.Wait()
}

func (w *Worker) startLoop(ctx context.Context) {
	w.wg.Add(1)

	go func() {
		timer := time.NewTimer(maxInactivityTimeout)

		defer func() {
			// notifies the caller that the worker has exited
			timer.Stop()
			w.wg.Done()
			w.doneCh <- w.key
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				current := uint64(time.Now().Unix())
				if time.Duration(current-w.lastProcessedRequest) > maxInactivityTimeout {
					return

				} else {
					if ok := timer.Reset(maxInactivityTimeout); ok {
						panic("resetting timer when it was already running")
					}
				}
			case arg, ok := <-w.inCh:
				if !ok {
					return
				}

				w.lastProcessedRequest = uint64(time.Now().Unix())
				w.serveRequest(arg)
			}
		}
	}()
}

func (w *Worker) serveRequest(req interface{}) {
	switch req := req.(type) {
	case insertRequest:
		w.insert(req)
	case retrieveRequest:
		w.retrieve(req)
	case discardRequest:
		w.discard(req)
	case nextRequest:
		w.next(req)
	default:
		panic("invalid request received for worker")
	}
}

func (w *Worker) insert(req insertRequest) {
	req.Out <- w.queue.Insert(req.Element)
}

func (w *Worker) retrieve(req retrieveRequest) {
	req.Out <- w.queue.Retrieve(req.Offset, req.Count)
}

func (w *Worker) discard(req discardRequest) {
	w.queue.Discard(req.Offset)
	req.Out <- nil
}

func (w *Worker) next(req nextRequest) {
	req.Out <- w.queue.Next()
}

func (w *Worker) Insert(req insertRequest) {
	w.inCh <- req
}

func (w *Worker) Retrieve(req retrieveRequest) {
	w.inCh <- req
}

func (w *Worker) Discard(req discardRequest) {
	w.inCh <- req
}

func (w *Worker) Next(req nextRequest) {
	w.inCh <- req
}