package awsauth

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"

	"github.com/smartystreets/assertions"
	"github.com/smartystreets/assertions/should"
)

func TestVersion4RequestPreparer_1(t *testing.T) {
	// Given a plain request with no custom headers
	request := test_plainRequestV4(false)
	prepareRequestV4(request)

	expectedUnsigned := test_unsignedRequestV4(true, false)
	expectedUnsigned.Header.Set("X-Amz-Date", timestampV4())

	assert := assertions.New(t)

	// The necessary, default headers should be appended
	assert.So(dumpRequest(request), should.Equal, dumpRequest(expectedUnsigned))

	// Forward-slash should be appended to URI if not present
	assert.So(request.URL.Path, should.Equal, "/")
}

func TestVersion4RequestPreparer_2(t *testing.T) {
	// And a set of credentials
	// It should be signed with an Authorization header
	request := test_plainRequestV4(false)
	actualSigned := Sign4(request, *testCredV4)
	actual := actualSigned.Header.Get("Authorization")

	assert := assertions.New(t)
	assert.So(actual, should.NotBeBlank)
	assert.So(actual, should.ContainSubstring, "Credential="+testCredV4.AccessKeyID)
	assert.So(actual, should.ContainSubstring, "SignedHeaders=")
	assert.So(actual, should.ContainSubstring, "Signature=")
	assert.So(actual, should.ContainSubstring, "AWS4")
}

func TestVersion4RequestPreparer_3(t *testing.T) {
	// Given a request with custom, necessary headers
	// The custom, necessary headers must not be changed
	request := test_unsignedRequestV4(true, false)
	prepareRequestV4(request)
	assertions.New(t).So(dumpRequest(request), should.Equal, dumpRequest(test_unsignedRequestV4(true, false)))
}

func TestVersion4STSRequestPreparer(t *testing.T) {
	// Given a plain request with no custom headers
	request := test_plainRequestV4(false)

	// And a set of credentials with an STS token
	var keys Credentials
	keys = *testCredV4WithSTS

	// It should include an X-Amz-Security-Token when the request is signed
	actualSigned := Sign4(request, keys)
	actual := actualSigned.Header.Get("X-Amz-Security-Token")

	assert := assertions.New(t)
	assert.So(actual, should.NotBeBlank)
	assert.So(actual, should.Equal, testCredV4WithSTS.SecurityToken)
}

func TestVersion4SigningTasks(t *testing.T) {
	// http://docs.aws.amazon.com/general/latest/gr/sigv4_signing.html

	// Given a bogus request and credentials from AWS documentation with an additional meta tag
	request := test_unsignedRequestV4(true, true)
	meta := new(metadata)
	assert := assertions.New(t)

	// (Task 1) The canonical request should be built correctly
	hashedCanonReq := hashedCanonicalRequestV4(request, meta)
	assert.So(hashedCanonReq, should.Equal, expectingV4["CanonicalHash"])

	// (Task 2) The string to sign should be built correctly
	stringToSign := stringToSignV4(request, hashedCanonReq, meta)
	assert.So(stringToSign, should.Equal, expectingV4["StringToSign"])

	// (Task 3) The version 4 signed signature should be correct
	signature := signatureV4(test_signingKeyV4(), stringToSign)
	assert.So(signature, should.Equal, expectingV4["SignatureV4"])
}

func TestSignature4Helpers(t *testing.T) {
	// The signing key should be properly generated
	expected := []byte{152, 241, 216, 137, 254, 196, 244, 66, 26, 220, 82, 43, 171, 12, 225, 248, 46, 105, 41, 194, 98, 237, 21, 229, 169, 76, 144, 239, 209, 227, 176, 231}
	actual := test_signingKeyV4()

	assertions.New(t).So(actual, should.Resemble, expected)
}
func TestSignature4Helpers_1(t *testing.T) {
	// Authorization headers should be built properly
	meta := &metadata{
		algorithm:       "AWS4-HMAC-SHA256",
		credentialScope: "20110909/us-east-1/iam/aws4_request",
		signedHeaders:   "content-type;host;x-amz-date",
	}
	expected := expectingV4["AuthHeader"] + expectingV4["SignatureV4"]
	actual := buildAuthHeaderV4(expectingV4["SignatureV4"], meta, *testCredV4)

	assertions.New(t).So(actual, should.Equal, expected)
}
func TestSignature4Helpers_2(t *testing.T) {
	// Timestamps should be in the correct format, in UTC time
	actual := timestampV4()

	assert := assertions.New(t)
	assert.So(len(actual), should.Equal, 16)
	assert.So(actual, should.NotContainSubstring, ":")
	assert.So(actual, should.NotContainSubstring, "-")
	assert.So(actual, should.NotContainSubstring, " ")
	assert.So(actual, should.EndWith, "Z")
	assert.So(actual, should.ContainSubstring, "T")
}
func TestSignature4Helpers_3(t *testing.T) {
	// Given an Version 4 AWS-formatted timestamp
	ts := "20110909T233600Z"

	// The date string should be extracted properly
	assertions.New(t).So(tsDateV4(ts), should.Equal, "20110909")
}
func TestSignature4Helpers_4(t *testing.T) {
	// Given any request with a body
	request := test_plainRequestV4(false)

	// Its body should be read and replaced without differences
	expected := []byte(requestValuesV4.Encode())
	assert := assertions.New(t)

	actual1 := readAndReplaceBody(request)
	assert.So(actual1, should.Resemble, expected)

	actual2 := readAndReplaceBody(request)
	assert.So(actual2, should.Resemble, expected)
}

func test_plainRequestV4(trailingSlash bool) *http.Request {
	address := "http://iam.amazonaws.com"
	body := strings.NewReader(requestValuesV4.Encode())

	if trailingSlash {
		address += "/"
	}

	request, err := http.NewRequest("POST", address, body)

	if err != nil {
		panic(err)
	}

	return request
}

func test_unsignedRequestV4(trailingSlash, tag bool) *http.Request {
	request := test_plainRequestV4(trailingSlash)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	request.Header.Set("X-Amz-Date", "20110909T233600Z")
	if tag {
		request.Header.Set("X-Amz-Meta-Foo", "Bar!")
	}
	return request
}

func test_signingKeyV4() []byte {
	return signingKeyV4(testCredV4.SecretAccessKey, "20110909", "us-east-1", "iam")
}

func dumpRequest(request *http.Request) string {
	dump, _ := httputil.DumpRequestOut(request, true)
	return string(dump)
}

var (
	testCredV4 = &Credentials{
		AccessKeyID:     "AKIDEXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
	}

	testCredV4WithSTS = &Credentials{
		AccessKeyID:     "AKIDEXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
		SecurityToken:   "AQoDYXdzEHcaoAJ1Aqwx1Sum0iW2NQjXJcWlKR7vuB6lnAeGBaQnjDRZPVyniwc48ml5hx+0qiXenVJdfusMMl9XLhSncfhx9Rb1UF8IAOaQ+CkpWXvoH67YYN+93dgckSVgVEBRByTl/BvLOZhe0ii/pOWkuQtBm5T7lBHRe4Dfmxy9X6hd8L3FrWxgnGV3fWZ3j0gASdYXaa+VBJlU0E2/GmCzn3T+t2mjYaeoInAnYVKVpmVMOrh6lNAeETTOHElLopblSa7TAmROq5xHIyu4a9i2qwjERTwa3Yk4Jk6q7JYVA5Cu7kS8wKVml8LdzzCTsy+elJgvH+Jf6ivpaHt/En0AJ5PZUJDev2+Y5+9j4AYfrmXfm4L73DC1ZJFJrv+Yh+EXAMPLE=",
	}

	expectingV4 = map[string]string{
		"CanonicalHash": "41c56ed0df12052f7c10407a809e64cd61a4b0471956cdea28d6d1bb904f5d92",
		"StringToSign":  "AWS4-HMAC-SHA256\n20110909T233600Z\n20110909/us-east-1/iam/aws4_request\n41c56ed0df12052f7c10407a809e64cd61a4b0471956cdea28d6d1bb904f5d92",
		"SignatureV4":   "08292a4b86aae1a6f80f1988182a33cbf73ccc70c5da505303e355a67cc64cb4",
		"AuthHeader":    "AWS4-HMAC-SHA256 Credential=AKIDEXAMPLE/20110909/us-east-1/iam/aws4_request, SignedHeaders=content-type;host;x-amz-date, Signature=",
	}

	requestValuesV4 = &url.Values{
		"Action":  []string{"ListUsers"},
		"Version": []string{"2010-05-08"},
	}
)
