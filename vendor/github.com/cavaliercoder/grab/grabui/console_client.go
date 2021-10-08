package grabui

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/cavaliercoder/grab"
)

type ConsoleClient struct {
	mu                            sync.Mutex
	client                        *grab.Client
	succeeded, failed, inProgress int
	responses                     []*grab.Response
}

func NewConsoleClient(client *grab.Client) *ConsoleClient {
	return &ConsoleClient{
		client: client,
	}
}

func (c *ConsoleClient) Do(
	ctx context.Context,
	workers int,
	reqs ...*grab.Request,
) <-chan *grab.Response {
	// buffer size prevents slow receivers causing back pressure
	pump := make(chan *grab.Response, len(reqs))

	go func() {
		c.mu.Lock()
		defer c.mu.Unlock()

		c.failed = 0
		c.inProgress = 0
		c.succeeded = 0
		c.responses = make([]*grab.Response, 0, len(reqs))
		if c.client == nil {
			c.client = grab.DefaultClient
		}

		fmt.Printf("Downloading %d files...\n", len(reqs))
		respch := c.client.DoBatch(workers, reqs...)
		t := time.NewTicker(200 * time.Millisecond)
		defer t.Stop()

	Loop:
		for {
			select {
			case <-ctx.Done():
				break Loop

			case resp := <-respch:
				if resp != nil {
					// a new response has been received and has started downloading
					c.responses = append(c.responses, resp)
					pump <- resp // send to caller
				} else {
					// channel is closed - all downloads are complete
					break Loop
				}

			case <-t.C:
				// update UI on clock tick
				c.refresh()
			}
		}

		c.refresh()
		close(pump)

		fmt.Printf(
			"Finished %d successful, %d failed, %d incomplete.\n",
			c.succeeded,
			c.failed,
			c.inProgress)
	}()
	return pump
}

// refresh prints the progress of all downloads to the terminal
func (c *ConsoleClient) refresh() {
	// clear lines for incomplete downloads
	if c.inProgress > 0 {
		fmt.Printf("\033[%dA\033[K", c.inProgress)
	}

	// print newly completed downloads
	for i, resp := range c.responses {
		if resp != nil && resp.IsComplete() {
			if resp.Err() != nil {
				c.failed++
				fmt.Fprintf(os.Stderr, "Error downloading %s: %v\n",
					resp.Request.URL(),
					resp.Err())
			} else {
				c.succeeded++
				fmt.Printf("Finished %s %s / %s (%d%%)\n",
					resp.Filename,
					byteString(resp.BytesComplete()),
					byteString(resp.Size()),
					int(100*resp.Progress()))
			}
			c.responses[i] = nil
		}
	}

	// print progress for incomplete downloads
	c.inProgress = 0
	for _, resp := range c.responses {
		if resp != nil {
			fmt.Printf("Downloading %s %s / %s (%d%%) - %s ETA: %s \033[K\n",
				resp.Filename,
				byteString(resp.BytesComplete()),
				byteString(resp.Size()),
				int(100*resp.Progress()),
				bpsString(resp.BytesPerSecond()),
				etaString(resp.ETA()))
			c.inProgress++
		}
	}
}

func bpsString(n float64) string {
	if n < 1e3 {
		return fmt.Sprintf("%.02fBps", n)
	}
	if n < 1e6 {
		return fmt.Sprintf("%.02fKB/s", n/1e3)
	}
	if n < 1e9 {
		return fmt.Sprintf("%.02fMB/s", n/1e6)
	}
	return fmt.Sprintf("%.02fGB/s", n/1e9)
}

func byteString(n int64) string {
	if n < 1<<10 {
		return fmt.Sprintf("%dB", n)
	}
	if n < 1<<20 {
		return fmt.Sprintf("%dKB", n>>10)
	}
	if n < 1<<30 {
		return fmt.Sprintf("%dMB", n>>20)
	}
	if n < 1<<40 {
		return fmt.Sprintf("%dGB", n>>30)
	}
	return fmt.Sprintf("%dTB", n>>40)
}

func etaString(eta time.Time) string {
	d := eta.Sub(time.Now())
	if d < time.Second {
		return "<1s"
	}
	// truncate to 1s resolution
	d /= time.Second
	d *= time.Second
	return d.String()
}
