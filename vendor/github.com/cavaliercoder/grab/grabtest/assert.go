package grabtest

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"testing"
)

func AssertHTTPResponseStatusCode(t *testing.T, resp *http.Response, expect int) (ok bool) {
	if resp.StatusCode != expect {
		t.Errorf("expected status code: %d, got: %d", expect, resp.StatusCode)
		return
	}
	ok = true
	return true
}

func AssertHTTPResponseHeader(t *testing.T, resp *http.Response, key, format string, a ...interface{}) (ok bool) {
	expect := fmt.Sprintf(format, a...)
	actual := resp.Header.Get(key)
	if actual != expect {
		t.Errorf("expected header %s: %s, got: %s", key, expect, actual)
		return
	}
	ok = true
	return
}

func AssertHTTPResponseContentLength(t *testing.T, resp *http.Response, n int64) (ok bool) {
	ok = true
	if resp.ContentLength != n {
		ok = false
		t.Errorf("expected header Content-Length: %d, got: %d", n, resp.ContentLength)
	}
	if !AssertHTTPResponseBodyLength(t, resp, n) {
		ok = false
	}
	return
}

func AssertHTTPResponseBodyLength(t *testing.T, resp *http.Response, n int64) (ok bool) {
	defer func() {
		if err := resp.Body.Close(); err != nil {
			panic(err)
		}
	}()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	if int64(len(b)) != n {
		ok = false
		t.Errorf("expected body length: %d, got: %d", n, len(b))
	}
	return
}

func MustHTTPNewRequest(method, url string, body io.Reader) *http.Request {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		panic(err)
	}
	return req
}

func MustHTTPDo(req *http.Request) *http.Response {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	return resp
}

func MustHTTPDoWithClose(req *http.Request) *http.Response {
	resp := MustHTTPDo(req)
	if _, err := io.Copy(ioutil.Discard, resp.Body); err != nil {
		panic(err)
	}
	if err := resp.Body.Close(); err != nil {
		panic(err)
	}
	return resp
}

func AssertSHA256Sum(t *testing.T, sum []byte, r io.Reader) (ok bool) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		panic(err)
	}
	computed := h.Sum(nil)
	ok = bytes.Equal(sum, computed)
	if !ok {
		t.Errorf(
			"expected checksum: %s, got: %s",
			MustHexEncodeString(sum),
			MustHexEncodeString(computed),
		)
	}
	return
}
