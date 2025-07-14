package services

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// TMDBRateLimiter manages TMDB API rate limiting using token bucket algorithm
// TMDB allows 50 requests per 10 seconds, we use 40 to be conservative
type TMDBRateLimiter struct {
	db                *sql.DB
	maxRequests       int           // Maximum requests per window
	windowDuration    time.Duration // Time window duration
	refillRate        time.Duration // How often to add tokens
	tokens            int           // Current available tokens
	lastRefill        time.Time     // Last time tokens were refilled
	mutex             sync.Mutex    // Thread safety
	requestQueue      chan *RateLimitRequest // Queue for pending requests
	isRunning         bool          // Whether the limiter is running
	stopChan          chan bool     // Channel to stop the limiter
}

// RateLimitRequest represents a pending API request
type RateLimitRequest struct {
	callback   func() error // Function to execute when rate limit allows
	resultChan chan error   // Channel to send result back
	priority   int          // Request priority (higher = more important)
	createdAt  time.Time    // When request was created
}

// NewTMDBRateLimiter creates a new TMDB rate limiter
func NewTMDBRateLimiter(db *sql.DB) *TMDBRateLimiter {
	limiter := &TMDBRateLimiter{
		db:             db,
		maxRequests:    40,                // 40 requests per 10 seconds (80% of TMDB limit)
		windowDuration: 10 * time.Second,  // 10 second window
		refillRate:     250 * time.Millisecond, // Refill every 250ms (40 tokens over 10s)
		tokens:         40,                // Start with full bucket
		lastRefill:     time.Now(),
		requestQueue:   make(chan *RateLimitRequest, 1000), // Buffer up to 1000 requests
		stopChan:       make(chan bool),
	}
	
	// Start the background processor
	go limiter.processRequests()
	
	return limiter
}

// ExecuteWithRateLimit executes a function with rate limiting
// Priority: 0 = low (background sync), 1 = normal (user requests), 2 = high (user-triggered)
func (r *TMDBRateLimiter) ExecuteWithRateLimit(fn func() error, priority int) error {
	request := &RateLimitRequest{
		callback:   fn,
		resultChan: make(chan error, 1),
		priority:   priority,
		createdAt:  time.Now(),
	}
	
	// Add to queue (this will block if queue is full)
	select {
	case r.requestQueue <- request:
		// Request queued successfully
	case <-time.After(30 * time.Second):
		return fmt.Errorf("rate limiter queue is full, request timed out")
	}
	
	// Wait for result
	select {
	case err := <-request.resultChan:
		return err
	case <-time.After(5 * time.Minute):
		return fmt.Errorf("rate limited request timed out after 5 minutes")
	}
}

// processRequests runs in background and processes queued requests
func (r *TMDBRateLimiter) processRequests() {
	r.isRunning = true
	refillTicker := time.NewTicker(r.refillRate)
	defer refillTicker.Stop()
	
	// Priority queue to handle high-priority requests first
	var pendingRequests []*RateLimitRequest
	
	for {
		select {
		case <-r.stopChan:
			r.isRunning = false
			return
			
		case <-refillTicker.C:
			r.refillTokens()
			
		case request := <-r.requestQueue:
			// Add to pending requests in priority order
			pendingRequests = r.insertByPriority(pendingRequests, request)
			
		default:
			// Process pending requests if we have tokens
			if len(pendingRequests) > 0 && r.hasTokens() {
				request := pendingRequests[0]
				pendingRequests = pendingRequests[1:]
				
				r.consumeToken()
				go r.executeRequest(request)
			} else {
				// Small sleep to prevent busy waiting
				time.Sleep(10 * time.Millisecond)
			}
		}
	}
}

// insertByPriority inserts request in correct priority order
func (r *TMDBRateLimiter) insertByPriority(requests []*RateLimitRequest, newRequest *RateLimitRequest) []*RateLimitRequest {
	// Find insertion point (higher priority first, then by creation time)
	insertAt := len(requests)
	for i, req := range requests {
		if newRequest.priority > req.priority || 
		   (newRequest.priority == req.priority && newRequest.createdAt.Before(req.createdAt)) {
			insertAt = i
			break
		}
	}
	
	// Insert at the correct position
	requests = append(requests, nil)
	copy(requests[insertAt+1:], requests[insertAt:])
	requests[insertAt] = newRequest
	return requests
}

// executeRequest executes a rate-limited request with retry logic
func (r *TMDBRateLimiter) executeRequest(request *RateLimitRequest) {
	var err error
	maxRetries := 3
	backoffDelay := 1 * time.Second
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			time.Sleep(backoffDelay)
			backoffDelay *= 2
		}
		
		err = request.callback()
		if err == nil {
			// Success
			r.recordSuccessfulRequest()
			request.resultChan <- nil
			return
		}
		
		// Check if it's a rate limit error that should be retried
		if r.shouldRetry(err) && attempt < maxRetries {
			fmt.Printf("TMDB API request failed (attempt %d/%d): %v\n", attempt+1, maxRetries+1, err)
			continue
		}
		
		// Max retries reached or non-retryable error
		break
	}
	
	// Request failed
	r.recordFailedRequest(err)
	request.resultChan <- err
}

// shouldRetry determines if an error should trigger a retry
func (r *TMDBRateLimiter) shouldRetry(err error) bool {
	if err == nil {
		return false
	}
	
	errStr := err.Error()
	// Retry on rate limit, timeout, or temporary network errors
	return contains(errStr, "rate limit") || 
		   contains(errStr, "timeout") || 
		   contains(errStr, "temporary failure") ||
		   contains(errStr, "connection reset")
}

// refillTokens adds tokens to the bucket based on time elapsed
func (r *TMDBRateLimiter) refillTokens() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	now := time.Now()
	elapsed := now.Sub(r.lastRefill)
	
	// Calculate tokens to add (1 token per 250ms)
	tokensToAdd := int(elapsed / r.refillRate)
	if tokensToAdd > 0 {
		r.tokens = min(r.maxRequests, r.tokens+tokensToAdd)
		r.lastRefill = now
	}
}

// hasTokens checks if tokens are available
func (r *TMDBRateLimiter) hasTokens() bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.tokens > 0
}

// consumeToken removes one token from the bucket
func (r *TMDBRateLimiter) consumeToken() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.tokens > 0 {
		r.tokens--
	}
}

// recordSuccessfulRequest logs successful API request
func (r *TMDBRateLimiter) recordSuccessfulRequest() {
	_, err := r.db.Exec(`
		UPDATE tmdb_rate_limits 
		SET requests_count = requests_count + 1, 
			last_request_at = datetime('now'),
			updated_at = datetime('now')
		WHERE id = 1
	`)
	if err != nil {
		fmt.Printf("Failed to record successful TMDB request: %v\n", err)
	}
}

// recordFailedRequest logs failed API request
func (r *TMDBRateLimiter) recordFailedRequest(requestErr error) {
	fmt.Printf("TMDB API request failed: %v\n", requestErr)
}

// GetStats returns current rate limiter statistics
func (r *TMDBRateLimiter) GetStats() map[string]interface{} {
	r.mutex.Lock()
	tokens := r.tokens
	queueSize := len(r.requestQueue)
	r.mutex.Unlock()
	
	var totalRequests int
	var lastRequest time.Time
	
	err := r.db.QueryRow(`
		SELECT requests_count, COALESCE(last_request_at, datetime('now')) 
		FROM tmdb_rate_limits WHERE id = 1
	`).Scan(&totalRequests, &lastRequest)
	
	if err != nil {
		fmt.Printf("Failed to get rate limit stats: %v\n", err)
	}
	
	return map[string]interface{}{
		"available_tokens": tokens,
		"max_tokens":      r.maxRequests,
		"queue_size":      queueSize,
		"total_requests":  totalRequests,
		"last_request":    lastRequest,
		"is_running":      r.isRunning,
	}
}

// Stop gracefully stops the rate limiter
func (r *TMDBRateLimiter) Stop() {
	if r.isRunning {
		r.stopChan <- true
	}
}

// Helper functions
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || (len(s) > len(substr) && 
		   	(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		   	 indexContains(s, substr) >= 0)))
}

func indexContains(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}