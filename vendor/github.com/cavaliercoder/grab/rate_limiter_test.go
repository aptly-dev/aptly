package grab

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/cavaliercoder/grab/grabtest"
)

// testRateLimiter is a naive rate limiter that limits throughput to r tokens
// per second. The total number of tokens issued is tracked as n.
type testRateLimiter struct {
	r, n int
}

func NewLimiter(r int) RateLimiter {
	return &testRateLimiter{r: r}
}

func (c *testRateLimiter) WaitN(ctx context.Context, n int) (err error) {
	c.n += n
	time.Sleep(
		time.Duration(1.00 / float64(c.r) * float64(n) * float64(time.Second)))
	return
}

func TestRateLimiter(t *testing.T) {
	// download a 128 byte file, 8 bytes at a time, with a naive 512bps limiter
	// should take > 250ms
	filesize := 128
	filename := ".testRateLimiter"
	defer os.Remove(filename)

	grabtest.WithTestServer(t, func(url string) {
		// limit to 512bps
		lim := &testRateLimiter{r: 512}
		req := mustNewRequest(filename, url)

		// ensure multiple trips to the rate limiter by downloading 8 bytes at a time
		req.BufferSize = 8
		req.RateLimiter = lim

		resp := mustDo(req)
		testComplete(t, resp)
		if lim.n != filesize {
			t.Errorf("expected %d bytes to pass through limiter, got %d", filesize, lim.n)
		}
		if resp.Duration().Seconds() < 0.25 {
			// BUG: this test can pass if the transfer was slow for unrelated reasons
			t.Errorf("expected transfer to take >250ms, took %v", resp.Duration())
		}
	}, grabtest.ContentLength(filesize))
}

func ExampleRateLimiter() {
	req, _ := NewRequest("", "http://www.golang-book.com/public/pdf/gobook.pdf")

	// Attach a 1Mbps rate limiter, like the token bucket implementation from
	// golang.org/x/time/rate.
	req.RateLimiter = NewLimiter(1048576)

	resp := DefaultClient.Do(req)
	if err := resp.Err(); err != nil {
		log.Fatal(err)
	}
}
