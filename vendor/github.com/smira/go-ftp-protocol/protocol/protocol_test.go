package protocol

import (
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

func TestProtocol(t *testing.T) {
	transport := &http.Transport{}
	transport.RegisterProtocol("ftp", &FTPRoundTripper{})

	client := &http.Client{Transport: transport}

	resp, err := client.Get("ftp://ftp.ru.debian.org/debian/README")
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 200 {
		t.Fatalf("resp.StatusCode 200 != %d", resp.StatusCode)
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	err = resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(string(content), "See http://www.debian.org/ for information about Debian GNU/Linux.") {
		t.Fatalf("unexpected content: %s", content)
	}

	resp, err = client.Get("ftp://ftp.ru.debian.org/debian/missing")
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 404 {
		t.Fatalf("resp.StatusCode 404 != %d", resp.StatusCode)
	}
}

func TestConcurrent(t *testing.T) {
	transport := &http.Transport{}
	transport.RegisterProtocol("ftp", &FTPRoundTripper{})

	client := &http.Client{Transport: transport}

	const concurrency = 4
	const count = 10

	done := make(chan struct{}, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer func() { done <- struct{}{} }()

			for j := 0; j < 10; j++ {
				resp, err := client.Get("ftp://ftp.ru.debian.org/debian/README")
				if err != nil {
					t.Fatal(err)
				}

				if resp.StatusCode != 200 {
					t.Fatalf("resp.StatusCode 200 != %d", resp.StatusCode)
				}

				content, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					t.Fatal(err)
				}

				err = resp.Body.Close()
				if err != nil {
					t.Fatal(err)
				}

				if !strings.HasPrefix(string(content), "See http://www.debian.org/ for information about Debian GNU/Linux.") {
					t.Fatalf("unexpected content: %s", content)
				}
			}
		}()
	}

	for i := 0; i < concurrency; i++ {
		<-done
	}
}
