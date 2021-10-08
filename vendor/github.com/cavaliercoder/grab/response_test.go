package grab

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/cavaliercoder/grab/grabtest"
)

// testComplete validates that a completed Response has all the desired fields.
func testComplete(t *testing.T, resp *Response) {
	<-resp.Done
	if !resp.IsComplete() {
		t.Errorf("Response.IsComplete returned false")
	}

	if resp.Start.IsZero() {
		t.Errorf("Response.Start is zero")
	}

	if resp.End.IsZero() {
		t.Error("Response.End is zero")
	}

	if eta := resp.ETA(); eta != resp.End {
		t.Errorf("Response.ETA is not equal to Response.End: %v", eta)
	}

	// the following fields should only be set if no error occurred
	if resp.Err() == nil {
		if resp.Filename == "" {
			t.Errorf("Response.Filename is empty")
		}

		if resp.Size() == 0 {
			t.Error("Response.Size is zero")
		}

		if p := resp.Progress(); p != 1.00 {
			t.Errorf("Response.Progress returned %v (%v/%v bytes), expected 1", p, resp.BytesComplete(), resp.Size())
		}
	}
}

// TestResponseProgress tests the functions which indicate the progress of an
// in-process file transfer.
func TestResponseProgress(t *testing.T) {
	filename := ".testResponseProgress"
	defer os.Remove(filename)

	sleep := 300 * time.Millisecond
	size := 1024 * 8 // bytes

	grabtest.WithTestServer(t, func(url string) {
		// request a slow transfer
		req := mustNewRequest(filename, url)
		resp := DefaultClient.Do(req)

		// make sure transfer has not started
		if resp.IsComplete() {
			t.Errorf("Transfer should not have started")
		}

		if p := resp.Progress(); p != 0 {
			t.Errorf("Transfer should not have started yet but progress is %v", p)
		}

		// wait for transfer to complete
		<-resp.Done

		// make sure transfer is complete
		if p := resp.Progress(); p != 1 {
			t.Errorf("Transfer is complete but progress is %v", p)
		}

		if s := resp.BytesComplete(); s != int64(size) {
			t.Errorf("Expected to transfer %v bytes, got %v", size, s)
		}
	},
		grabtest.TimeToFirstByte(sleep),
		grabtest.ContentLength(size),
	)
}

func TestResponseOpen(t *testing.T) {
	grabtest.WithTestServer(t, func(url string) {
		resp := mustDo(mustNewRequest("", url+"/someFilename"))
		f, err := resp.Open()
		if err != nil {
			t.Error(err)
			return
		}
		defer func() {
			if err := f.Close(); err != nil {
				t.Error(err)
			}
		}()
		grabtest.AssertSHA256Sum(t, grabtest.DefaultHandlerSHA256ChecksumBytes, f)
	})
}

func TestResponseBytes(t *testing.T) {
	grabtest.WithTestServer(t, func(url string) {
		resp := mustDo(mustNewRequest("", url+"/someFilename"))
		b, err := resp.Bytes()
		if err != nil {
			t.Error(err)
			return
		}
		grabtest.AssertSHA256Sum(
			t,
			grabtest.DefaultHandlerSHA256ChecksumBytes,
			bytes.NewReader(b),
		)
	})
}
