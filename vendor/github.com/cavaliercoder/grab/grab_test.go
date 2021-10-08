package grab

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/cavaliercoder/grab/grabtest"
)

func TestMain(m *testing.M) {
	os.Exit(func() int {
		// chdir to temp so test files downloaded to pwd are isolated and cleaned up
		cwd, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		tmpDir, err := ioutil.TempDir("", "grab-")
		if err != nil {
			panic(err)
		}
		if err := os.Chdir(tmpDir); err != nil {
			panic(err)
		}
		defer func() {
			os.Chdir(cwd)
			if err := os.RemoveAll(tmpDir); err != nil {
				panic(err)
			}
		}()
		return m.Run()
	}())
}

// TestGet tests grab.Get
func TestGet(t *testing.T) {
	filename := ".testGet"
	defer os.Remove(filename)
	grabtest.WithTestServer(t, func(url string) {
		resp, err := Get(filename, url)
		if err != nil {
			t.Fatalf("error in Get(): %v", err)
		}
		testComplete(t, resp)
	})
}

func ExampleGet() {
	// download a file to /tmp
	resp, err := Get("/tmp", "http://example.com/example.zip")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Download saved to", resp.Filename)
}

func mustNewRequest(dst, urlStr string) *Request {
	req, err := NewRequest(dst, urlStr)
	if err != nil {
		panic(err)
	}
	return req
}

func mustDo(req *Request) *Response {
	resp := DefaultClient.Do(req)
	if err := resp.Err(); err != nil {
		panic(err)
	}
	return resp
}
