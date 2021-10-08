package grab

import (
	"fmt"
	"sync"
)

func ExampleClient_Do() {
	client := NewClient()
	req, err := NewRequest("/tmp", "http://example.com/example.zip")
	if err != nil {
		panic(err)
	}

	resp := client.Do(req)
	if err := resp.Err(); err != nil {
		panic(err)
	}

	fmt.Println("Download saved to", resp.Filename)
}

// This example uses DoChannel to create a Producer/Consumer model for
// downloading multiple files concurrently. This is similar to how DoBatch uses
// DoChannel under the hood except that it allows the caller to continually send
// new requests until they wish to close the request channel.
func ExampleClient_DoChannel() {
	// create a request and a buffered response channel
	reqch := make(chan *Request)
	respch := make(chan *Response, 10)

	// start 4 workers
	client := NewClient()
	wg := sync.WaitGroup{}
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			client.DoChannel(reqch, respch)
			wg.Done()
		}()
	}

	go func() {
		// send requests
		for i := 0; i < 10; i++ {
			url := fmt.Sprintf("http://example.com/example%d.zip", i+1)
			req, err := NewRequest("/tmp", url)
			if err != nil {
				panic(err)
			}
			reqch <- req
		}
		close(reqch)

		// wait for workers to finish
		wg.Wait()
		close(respch)
	}()

	// check each response
	for resp := range respch {
		// block until complete
		if err := resp.Err(); err != nil {
			panic(err)
		}

		fmt.Printf("Downloaded %s to %s\n", resp.Request.URL(), resp.Filename)
	}
}

func ExampleClient_DoBatch() {
	// create multiple download requests
	reqs := make([]*Request, 0)
	for i := 0; i < 10; i++ {
		url := fmt.Sprintf("http://example.com/example%d.zip", i+1)
		req, err := NewRequest("/tmp", url)
		if err != nil {
			panic(err)
		}
		reqs = append(reqs, req)
	}

	// start downloads with 4 workers
	client := NewClient()
	respch := client.DoBatch(4, reqs...)

	// check each response
	for resp := range respch {
		if err := resp.Err(); err != nil {
			panic(err)
		}

		fmt.Printf("Downloaded %s to %s\n", resp.Request.URL(), resp.Filename)
	}
}
