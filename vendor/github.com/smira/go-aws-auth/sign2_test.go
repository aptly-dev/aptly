package awsauth

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/smartystreets/assertions"
	"github.com/smartystreets/assertions/should"
	"github.com/smartystreets/gunit"
)

// http://docs.aws.amazon.com/general/latest/gr/signature-version-2.html

func TestSignature2Fixture(t *testing.T) {
	gunit.RunSequential(new(Signature2Fixture), t)
}

type Signature2Fixture struct {
	*gunit.Fixture

	keys Credentials
}

func (this *Signature2Fixture) Setup() {
	this.keys = *testCredV2

	// Mock time
	now = func() time.Time {
		parsed, _ := time.Parse(timeFormatV2, exampleReqTsV2)
		return parsed
	}
}

func (this *Signature2Fixture) TestSignUnpreparedPlanRequest() {
	request := test_plainRequestV2()
	prepareRequestV2(request, this.keys)
	this.So(request, should.Resemble, test_unsignedRequestV2())
}

func (this *Signature2Fixture) TestSignPreparedUnsignedRequest() {
	request := test_unsignedRequestV2()
	actual := canonicalQueryStringV2(request)
	expected := canonicalQsV2
	this.So(actual, should.Equal, expected)
	this.So(request.URL.Path, should.Equal, "/")

	this.So(stringToSignV2(request), should.Equal, expectedStringToSignV2)
	this.So(signatureV2(stringToSignV2(request), this.keys), should.Equal, "i91nKc4PWAt0JJIdXwz9HxZCJDdiy6cf/Mj6vPxyYIs=")

	Sign2(request, this.keys)
	this.So(request.URL.String(), should.Equal, expectedFinalUrlV2)
}

func TestVersion2STSRequestPreparer(t *testing.T) {
	// Given a plain request
	request := test_plainRequestV2()

	// And a set of credentials with an STS token
	var keys Credentials
	keys = *testCredV2WithSTS

	// It should include the SecurityToken parameter when the request is signed
	actualSigned := Sign2(request, keys)
	actual := actualSigned.URL.Query()["SecurityToken"][0]

	assert := assertions.New(t)
	assert.So(actual, should.NotBeBlank)
	assert.So(actual, should.Equal, testCredV2WithSTS.SecurityToken)
}

func test_plainRequestV2() *http.Request {
	values := url.Values{}
	values.Set("Action", "DescribeJobFlows")
	values.Set("Version", "2009-03-31")

	address := baseUrlV2 + "?" + values.Encode()

	request, err := http.NewRequest("GET", address, nil)
	if err != nil {
		panic(err)
	}

	return request
}

func test_unsignedRequestV2() *http.Request {
	request := test_plainRequestV2()
	newUrl, _ := url.Parse(baseUrlV2 + "/?" + canonicalQsV2)
	request.URL = newUrl
	return request
}

var (
	testCredV2 = &Credentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}

	testCredV2WithSTS = &Credentials{
		AccessKeyID:     "AKIDEXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
		SecurityToken:   "AQoDYXdzEHcaoAJ1Aqwx1Sum0iW2NQjXJcWlKR7vuB6lnAeGBaQnjDRZPVyniwc48ml5hx+0qiXenVJdfusMMl9XLhSncfhx9Rb1UF8IAOaQ+CkpWXvoH67YYN+93dgckSVgVEBRByTl/BvLOZhe0ii/pOWkuQtBm5T7lBHRe4Dfmxy9X6hd8L3FrWxgnGV3fWZ3j0gASdYXaa+VBJlU0E2/GmCzn3T+t2mjYaeoInAnYVKVpmVMOrh6lNAeETTOHElLopblSa7TAmROq5xHIyu4a9i2qwjERTwa3Yk4Jk6q7JYVA5Cu7kS8wKVml8LdzzCTsy+elJgvH+Jf6ivpaHt/En0AJ5PZUJDev2+Y5+9j4AYfrmXfm4L73DC1ZJFJrv+Yh+EXAMPLE=",
	}

	exampleReqTsV2         = "2011-10-03T15:19:30"
	baseUrlV2              = "https://elasticmapreduce.amazonaws.com"
	canonicalQsV2          = "AWSAccessKeyId=AKIAIOSFODNN7EXAMPLE&Action=DescribeJobFlows&SignatureMethod=HmacSHA256&SignatureVersion=2&Timestamp=2011-10-03T15%3A19%3A30&Version=2009-03-31"
	expectedStringToSignV2 = "GET\nelasticmapreduce.amazonaws.com\n/\n" + canonicalQsV2
	expectedFinalUrlV2     = baseUrlV2 + "/?AWSAccessKeyId=AKIAIOSFODNN7EXAMPLE&Action=DescribeJobFlows&Signature=i91nKc4PWAt0JJIdXwz9HxZCJDdiy6cf%2FMj6vPxyYIs%3D&SignatureMethod=HmacSHA256&SignatureVersion=2&Timestamp=2011-10-03T15%3A19%3A30&Version=2009-03-31"
)
