package grab

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

func TestURLFilenames(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		expect := "filename"
		testCases := []string{
			"http://test.com/filename",
			"http://test.com/path/filename",
			"http://test.com/deep/path/filename",
			"http://test.com/filename?with=args",
			"http://test.com/filename#with-fragment",
			"http://test.com/filename?with=args&and#with-fragment",
		}

		for _, tc := range testCases {
			req, _ := http.NewRequest("GET", tc, nil)
			resp := &http.Response{
				Request: req,
			}
			actual, err := guessFilename(resp)
			if err != nil {
				t.Errorf("%v", err)
			}

			if actual != expect {
				t.Errorf("expected '%v', got '%v'", expect, actual)
			}
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		testCases := []string{
			"http://test.com",
			"http://test.com/",
			"http://test.com/filename/",
			"http://test.com/filename/?with=args",
			"http://test.com/filename/#with-fragment",
			"http://test.com/filename\x00",
		}

		for _, tc := range testCases {
			t.Run(tc, func(t *testing.T) {
				req, err := http.NewRequest("GET", tc, nil)
				if err != nil {
					if tc == "http://test.com/filename\x00" {
						// Since go1.12, urls with invalid control character return an error
						// See https://github.com/golang/go/commit/829c5df58694b3345cb5ea41206783c8ccf5c3ca
						t.Skip()
					}
				}
				resp := &http.Response{
					Request: req,
				}

				_, err = guessFilename(resp)
				if err != ErrNoFilename {
					t.Errorf("expected '%v', got '%v'", ErrNoFilename, err)
				}
			})
		}
	})
}

func TestHeaderFilenames(t *testing.T) {
	u, _ := url.ParseRequestURI("http://test.com/badfilename")
	resp := &http.Response{
		Request: &http.Request{
			URL: u,
		},
		Header: http.Header{},
	}

	setFilename := func(resp *http.Response, filename string) {
		resp.Header.Set("Content-Disposition", fmt.Sprintf("attachment;filename=\"%s\"", filename))
	}

	t.Run("Valid", func(t *testing.T) {
		expect := "filename"
		testCases := []string{
			"filename",
			"path/filename",
			"/path/filename",
			"../../filename",
			"/path/../../filename",
			"/../../././///filename",
		}

		for _, tc := range testCases {
			setFilename(resp, tc)
			actual, err := guessFilename(resp)
			if err != nil {
				t.Errorf("error (%v): %v", tc, err)
			}

			if actual != expect {
				t.Errorf("expected '%v' (%v), got '%v'", expect, tc, actual)
			}
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		testCases := []string{
			"",
			"/",
			".",
			"/.",
			"/./",
			"..",
			"../",
			"/../",
			"/path/",
			"../path/",
			"filename\x00",
			"filename/",
			"filename//",
			"filename/..",
		}

		for _, tc := range testCases {
			setFilename(resp, tc)
			if actual, err := guessFilename(resp); err != ErrNoFilename {
				t.Errorf("expected: %v (%v), got: %v (%v)", ErrNoFilename, tc, err, actual)
			}
		}
	})
}

func TestHeaderWithMissingDirective(t *testing.T) {
	u, _ := url.ParseRequestURI("http://test.com/filename")
	resp := &http.Response{
		Request: &http.Request{
			URL: u,
		},
		Header: http.Header{},
	}

	setHeader := func(resp *http.Response, value string) {
		resp.Header.Set("Content-Disposition", value)
	}

	t.Run("Valid", func(t *testing.T) {
		expect := "filename"
		testCases := []string{
			"inline",
			"attachment",
		}

		for _, tc := range testCases {
			setHeader(resp, tc)
			actual, err := guessFilename(resp)
			if err != nil {
				t.Errorf("error (%v): %v", tc, err)
			}

			if actual != expect {
				t.Errorf("expected '%v' (%v), got '%v'", expect, tc, actual)
			}
		}
	})
}
