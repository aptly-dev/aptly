package awsauth

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"net/http/httptest"

	"github.com/smartystreets/assertions"
	"github.com/smartystreets/assertions/should"
	"github.com/smartystreets/gunit"
)

// http://docs.aws.amazon.com/AmazonS3/2006-03-01/dev/RESTAuthentication.html
// Note: S3 now supports signed signature version 4
// (but signed URL requests still utilize a lot of the same functionality)

func TestSignatureS3Fixture(t *testing.T) {
	gunit.RunSequential(new(SignatureS3Fixture), t)
}

type SignatureS3Fixture struct {
	*gunit.Fixture

	keys    Credentials
	request *http.Request
}

func (this *SignatureS3Fixture) Setup() {
	this.keys = *testCredS3
	this.request = test_plainRequestS3()

	now = func() time.Time {
		parsed, _ := time.Parse(timeFormatS3, exampleReqTsS3)
		return parsed
	}
}

func (this *SignatureS3Fixture) TestRequestShouldHaveADateHeader() {
	prepareRequestS3(this.request)
	this.So(this.request.Header.Get("Date"), should.Equal, exampleReqTsS3)
}

func (this *SignatureS3Fixture) TestRequestShouldHaveCanonicalizedAmzHeaders() {
	req2 := test_headerRequestS3()
	actual := canonicalAmzHeadersS3(req2)
	this.So(actual, should.Equal, expectedCanonAmzHeadersS3)
}

func (this *SignatureS3Fixture) TestCanonicalizedResourceBuiltProperly() {
	actual := canonicalResourceS3(this.request)
	this.So(actual, should.Equal, expectedCanonResourceS3)
}

func (this *SignatureS3Fixture) TestStringToSignShouldBeCorrect() {
	actual := stringToSignS3(this.request)
	this.So(actual, should.Equal, expectedStringToSignS3)
}

func (this *SignatureS3Fixture) TestFinalSignatureShouldBeExactlyCorrect() {
	actual := signatureS3(stringToSignS3(this.request), this.keys)
	this.So(actual, should.Equal, "bWq2s1WEIj+Ydj0vQ697zp+IXMU=")
}

func (this *SignatureS3Fixture) TestQueryStringAuthentication() {
	this.request = httptest.NewRequest("GET", "https://johnsmith.s3.amazonaws.com/johnsmith/photos/puppy.jpg", nil)

	// The string to sign should be correct
	actual := stringToSignS3Url("GET", now(), this.request.URL.Path)
	this.So(actual, should.Equal, expectedStringToSignS3Url)

	// The signature of string to sign should be correct
	actualSignature := signatureS3(expectedStringToSignS3Url, this.keys)
	this.So(actualSignature, should.Equal, "R2K/+9bbnBIbVDCs7dqlz3XFtBQ=")

	// The finished signed URL should be correct
	expiry := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	this.So(SignS3Url(this.request, expiry, this.keys).URL.String(), should.Equal, expectedSignedS3Url)
}

func TestS3STSRequestPreparer(t *testing.T) {
	// Given a plain request with no custom headers
	request := test_plainRequestS3()

	// And a set of credentials with an STS token
	keys := *testCredS3WithSTS

	// It should include an X-Amz-Security-Token when the request is signed
	actualSigned := SignS3(request, keys)
	actual := actualSigned.Header.Get("X-Amz-Security-Token")

	assert := assertions.New(t)
	assert.So(actual, should.NotBeBlank)
	assert.So(actual, should.Equal, testCredS3WithSTS.SecurityToken)
}

func test_plainRequestS3() *http.Request {
	return httptest.NewRequest("GET", "https://johnsmith.s3.amazonaws.com/photos/puppy.jpg", nil)
}

func test_headerRequestS3() *http.Request {
	request := test_plainRequestS3()
	request.Header.Set("X-Amz-Meta-Something", "more foobar")
	request.Header.Set("X-Amz-Date", "foobar")
	request.Header.Set("X-Foobar", "nanoo-nanoo")
	return request
}

func TestCanonical(t *testing.T) {
	expectedCanonicalString := "PUT\nc8fdb181845a4ca6b8fec737b3581d76\ntext/html\nThu, 17 Nov 2005 18:49:58 GMT\nx-amz-magic:abracadabra\nx-amz-meta-author:foo@bar.com\n/quotes/nelson"

	origUrl := "https://s3.amazonaws.com/"
	resource := "/quotes/nelson"

	u, _ := url.ParseRequestURI(origUrl)
	u.Path = resource
	urlStr := fmt.Sprintf("%v", u)

	request, _ := http.NewRequest("PUT", urlStr, nil)
	request.Header.Add("Content-Md5", "c8fdb181845a4ca6b8fec737b3581d76")
	request.Header.Add("Content-Type", "text/html")
	request.Header.Add("Date", "Thu, 17 Nov 2005 18:49:58 GMT")
	request.Header.Add("X-Amz-Meta-Author", "foo@bar.com")
	request.Header.Add("X-Amz-Magic", "abracadabra")

	if stringToSignS3(request) != expectedCanonicalString {
		t.Errorf("----Got\n***%s***\n----Expected\n***%s***", stringToSignS3(request), expectedCanonicalString)
	}
}

var (
	testCredS3 = &Credentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}

	testCredS3WithSTS = &Credentials{
		AccessKeyID:     "AKIDEXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
		SecurityToken:   "AQoDYXdzEHcaoAJ1Aqwx1Sum0iW2NQjXJcWlKR7vuB6lnAeGBaQnjDRZPVyniwc48ml5hx+0qiXenVJdfusMMl9XLhSncfhx9Rb1UF8IAOaQ+CkpWXvoH67YYN+93dgckSVgVEBRByTl/BvLOZhe0ii/pOWkuQtBm5T7lBHRe4Dfmxy9X6hd8L3FrWxgnGV3fWZ3j0gASdYXaa+VBJlU0E2/GmCzn3T+t2mjYaeoInAnYVKVpmVMOrh6lNAeETTOHElLopblSa7TAmROq5xHIyu4a9i2qwjERTwa3Yk4Jk6q7JYVA5Cu7kS8wKVml8LdzzCTsy+elJgvH+Jf6ivpaHt/En0AJ5PZUJDev2+Y5+9j4AYfrmXfm4L73DC1ZJFJrv+Yh+EXAMPLE=",
	}

	expectedCanonAmzHeadersS3 = "x-amz-date:foobar\nx-amz-meta-something:more foobar\n"
	expectedCanonResourceS3   = "/johnsmith/photos/puppy.jpg"
	expectedStringToSignS3    = "GET\n\n\nTue, 27 Mar 2007 19:36:42 +0000\n/johnsmith/photos/puppy.jpg"
	expectedStringToSignS3Url = "GET\n\n\n1175024202\n/johnsmith/photos/puppy.jpg"
	expectedSignedS3Url       = "https://johnsmith.s3.amazonaws.com/johnsmith/photos/puppy.jpg?AWSAccessKeyId=AKIAIOSFODNN7EXAMPLE&Expires=1257894000&Signature=X%2FarTLAJP08uP1Bsap52rwmsVok%3D"
	exampleReqTsS3            = "Tue, 27 Mar 2007 19:36:42 +0000"
)
