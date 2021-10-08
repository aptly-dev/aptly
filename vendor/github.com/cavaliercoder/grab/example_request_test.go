package grab

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

func ExampleRequest_WithContext() {
	// create context with a 100ms timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// create download request with context
	req, err := NewRequest("", "http://example.com/example.zip")
	if err != nil {
		panic(err)
	}
	req = req.WithContext(ctx)

	// send download request
	resp := DefaultClient.Do(req)
	if err := resp.Err(); err != nil {
		fmt.Println("error: request cancelled")
	}

	// Output:
	// error: request cancelled
}

func ExampleRequest_SetChecksum() {
	// create download request
	req, err := NewRequest("", "http://example.com/example.zip")
	if err != nil {
		panic(err)
	}

	// set request checksum
	sum, err := hex.DecodeString("33daf4c03f86120fdfdc66bddf6bfff4661c7ca11c5da473e537f4d69b470e57")
	if err != nil {
		panic(err)
	}
	req.SetChecksum(sha256.New(), sum, true)

	// download and validate file
	resp := DefaultClient.Do(req)
	if err := resp.Err(); err != nil {
		panic(err)
	}
}
