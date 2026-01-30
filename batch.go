package tryl

import (
	"context"
	"errors"
	"sync"
	"time"
)

// pendingEvent tracks an event and its result channel.
type pendingEvent struct {
	ctx      context.Context
	event    Event
	resultCh chan<- AsyncResult
	index    int
}

// Batcher accumulates events and sends them in batches.
type Batcher struct {
	client *Client
	config *BatchConfig

	pending chan pendingEvent
	stopCh  chan struct{}
	doneCh  chan struct{}

	mu      sync.Mutex
	stopped bool
}

// newBatcher creates a new Batcher.
func newBatcher(client *Client, config *BatchConfig) *Batcher {
	if config == nil {
		config = defaultBatchConfig()
	}
	if config.MaxBatchSize <= 0 {
		config.MaxBatchSize = 100
	}
	if config.FlushInterval <= 0 {
		config.FlushInterval = 5 * time.Second
	}
	if config.MaxPendingEvents <= 0 {
		config.MaxPendingEvents = 10000
	}

	b := &Batcher{
		client:  client,
		config:  config,
		pending: make(chan pendingEvent, config.MaxPendingEvents),
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
	}

	go b.run()

	return b
}

// Add queues an event for batching.
func (b *Batcher) Add(ctx context.Context, event Event, resultCh chan<- AsyncResult) {
	b.mu.Lock()
	if b.stopped {
		b.mu.Unlock()
		resultCh <- AsyncResult{Error: errors.New("batcher is stopped")}
		close(resultCh)
		return
	}
	b.mu.Unlock()

	select {
	case b.pending <- pendingEvent{ctx: ctx, event: event, resultCh: resultCh}:
	case <-ctx.Done():
		resultCh <- AsyncResult{Error: ctx.Err()}
		close(resultCh)
	}
}

// Flush sends all pending events immediately.
func (b *Batcher) Flush(ctx context.Context) error {
	var batch []pendingEvent

	for {
		select {
		case pe := <-b.pending:
			batch = append(batch, pe)
			if len(batch) >= b.config.MaxBatchSize {
				if err := b.sendBatch(ctx, batch); err != nil {
					return err
				}
				batch = nil
			}
		default:
			if len(batch) > 0 {
				return b.sendBatch(ctx, batch)
			}
			return nil
		}
	}
}

// Stop stops the batcher, flushing pending events.
func (b *Batcher) Stop(ctx context.Context) error {
	b.mu.Lock()
	if b.stopped {
		b.mu.Unlock()
		return nil
	}
	b.stopped = true
	b.mu.Unlock()

	close(b.stopCh)

	select {
	case <-b.doneCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// run is the background loop that processes batches.
func (b *Batcher) run() {
	defer close(b.doneCh)

	ticker := time.NewTicker(b.config.FlushInterval)
	defer ticker.Stop()

	var batch []pendingEvent

	for {
		select {
		case pe := <-b.pending:
			batch = append(batch, pe)

			if len(batch) >= b.config.MaxBatchSize {
				b.sendBatch(context.Background(), batch)
				batch = nil
			}

		case <-ticker.C:
			if len(batch) > 0 {
				b.sendBatch(context.Background(), batch)
				batch = nil
			}

		case <-b.stopCh:
			for {
				select {
				case pe := <-b.pending:
					batch = append(batch, pe)
				default:
					if len(batch) > 0 {
						b.sendBatch(context.Background(), batch)
					}
					return
				}
			}
		}
	}
}

// sendBatch sends a batch of events to the API.
func (b *Batcher) sendBatch(ctx context.Context, batch []pendingEvent) error {
	if len(batch) == 0 {
		return nil
	}

	events := make([]Event, len(batch))
	for i, pe := range batch {
		events[i] = pe.event
		batch[i].index = i
	}

	resp, err := b.client.LogBatch(ctx, events)

	if err != nil {
		for _, pe := range batch {
			pe.resultCh <- AsyncResult{Error: err}
			close(pe.resultCh)
		}
		if b.config.OnError != nil {
			b.config.OnError(events, err)
		}
		return err
	}

	// Map results by index since API returns results in order
	resultMap := make(map[int]*EventResponse)
	for i, r := range resp.Results {
		// Use the batch item's original index
		if i < len(batch) {
			resultMap[batch[i].index] = &EventResponse{ID: r.ID, Timestamp: r.Timestamp}
		}
	}

	errorMap := make(map[int]error)
	for _, e := range resp.Errors {
		errorMap[e.Index] = &APIError{
			HTTPStatus: 400,
			Code:       e.Code,
			Message:    e.Message,
		}
	}

	for i, pe := range batch {
		if err, ok := errorMap[i]; ok {
			pe.resultCh <- AsyncResult{Error: err}
		} else if i < len(resp.Results) {
			pe.resultCh <- AsyncResult{Response: &resp.Results[i]}
		} else {
			pe.resultCh <- AsyncResult{Error: errors.New("missing response for event")}
		}
		close(pe.resultCh)
	}

	return nil
}
