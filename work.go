package main

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// archiveWorker consume page IDs and calls archivePage respecting a global
// rate limiter (tickCh). Stops on ctx.Done() or when the channel is closed.
func archiveWorker(
	ctx context.Context,
	wg *sync.WaitGroup,
	client *http.Client,
	token string,
	ids <-chan string,
	tickCh <-chan time.Time,
	total *int64,
	errCh chan<- error,
	totalPages int,
	startedAt time.Time,
) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case id, ok := <-ids:
			if !ok {
				return
			}
			// Wait for the next tick (<= 3 req/s) or cancellation
			select {
			case <-ctx.Done():
				return
			case <-tickCh:
			}
			if err := archivePage(client, token, id); err != nil {
				// Emit the error only once
				select {
				case errCh <- err:
				default:
				}
				return
			}
			// Increment and broadcast progress
			n := atomic.AddInt64(total, 1)
			if n%100 == 0 {
				fmt.Printf("Archivé: %d…\n", n)
			}
			elapsed := time.Since(startedAt).Seconds()
			rate := 0.0
			if elapsed > 0 {
				rate = float64(n) / elapsed
			}
			remaining := float64(totalPages) - float64(n)
			eta := 0
			if rate > 0 && remaining > 0 {
				eta = int(remaining / rate)
			}
			// Blocked sending: ensures delivery to the hub after each deletion
			hub.broadcast <- Progress{Running: true, Deleted: n, Total: totalPages, EtaSeconds: eta}
		}
	}
}

// runDeletion execute the archiving of all pages in the database.
// Returns the number of archived items and any error encountered.
func runDeletion(ctx context.Context, token, databaseID string) (int64, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	throttle := time.NewTicker(350 * time.Millisecond) // ~3 req/s (global)
	defer throttle.Stop()

	// Channel of IDs to archive and synchronization
	ids := make(chan string, 200)
	var total int64
	errCh := make(chan error, 1)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 1) Preloading: paginate and collect all IDs to know the total
	var allIDs []string
	cursor := ""
	for {
		qr, err := queryDatabase(client, token, databaseID, cursor)
		if err != nil {
			fmt.Println("Query error:", err)
			cancel()
			break
		}
		if len(qr.Results) == 0 {
			break
		}
		for _, r := range qr.Results {
			allIDs = append(allIDs, r.ID)
		}
		if !qr.HasMore {
			break
		}
		cursor = qr.NextCursor
	}

	// Display the total before starting the actual archiving and notify the hub
	totalPages := len(allIDs)
	fmt.Printf("Total pages to archive: %d\n", totalPages)
	// Ensure the initial state is delivered (blocking send)
	hub.broadcast <- Progress{Running: true, Deleted: 0, Total: totalPages, EtaSeconds: 0}

	// worker pool (launched after having the total for ETA)
	nWorkers := 4
	if cpus := runtime.NumCPU(); cpus > nWorkers {
		nWorkers = cpus
	}
	var wg sync.WaitGroup
	wg.Add(nWorkers)
	startedAt := time.Now()
	for i := 0; i < nWorkers; i++ {
		go archiveWorker(ctx, &wg, client, token, ids, throttle.C, &total, errCh, totalPages, startedAt)
	}

	// quickly cancel the production in case of worker errors
	var firstErr atomic.Value // stores error
	go func() {
		if err := <-errCh; err != nil {
			firstErr.Store(err)
			cancel()
		}
	}()

	// 2) Production: push the IDs into the queue for the workers
	for _, id := range allIDs {
		select {
		case <-ctx.Done():
			break
		case ids <- id:
		}
	}

	// Close the channel and wait for all workers to finish
	close(ids)
	wg.Wait()

	if v := firstErr.Load(); v != nil {
		done := atomic.LoadInt64(&total)
		// Ensure final state is delivered even on error
		hub.broadcast <- Progress{Running: false, Deleted: done, Total: totalPages, EtaSeconds: 0}
		return done, v.(error)
	}
	done := atomic.LoadInt64(&total)
	fmt.Printf("Done. Archived pages: %d\n", done)
	// Ensure final state is delivered on success
	hub.broadcast <- Progress{Running: false, Deleted: done, Total: totalPages, EtaSeconds: 0}
	return done, nil
}
