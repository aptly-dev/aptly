package awsauth

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/smartystreets/assertions"
	"github.com/smartystreets/assertions/should"
	"github.com/smartystreets/gunit"
)

func TestIntegrationFixture(t *testing.T) {
	if !credentialsSet() {
		t.Skip("Required credentials absent from environment.")
	}

	gunit.RunSequential(new(IntegrationFixture), t)
}

type IntegrationFixture struct {
	*gunit.Fixture
}

func (this *IntegrationFixture) assertOK(response *http.Response) {
	if !this.So(response.StatusCode, should.Equal, http.StatusOK) {
		message, _ := ioutil.ReadAll(response.Body)
		this.Error(string(message))
	}
}

func (this *IntegrationFixture) LongTestSign4_IAM_OutOfOrderQueryString() {
	request := newRequest("GET", "https://iam.amazonaws.com/?Version=2010-05-08&Action=ListRoles", nil)
	response := sign4AndDo(request)
	this.assertOK(response)
}

func (this *IntegrationFixture) LongTestSign4_S3() {
	request, _ := http.NewRequest("GET", "https://s3.amazonaws.com", nil)
	response := sign4AndDo(request)
	this.assertOK(response)
}

func (this *IntegrationFixture) LongTestSign2_EC2() {
	request := newRequest("GET", "https://ec2.amazonaws.com/?Version=2013-10-15&Action=DescribeInstances", nil)
	response := sign2AndDo(request)
	this.assertOK(response)
}
func (this *IntegrationFixture) LongTestSign4_SQS() {
	request := newRequest("POST", "https://sqs.us-west-2.amazonaws.com", url.Values{"Action": []string{"ListQueues"}})
	response := sign4AndDo(request)
	this.assertOK(response)
}

func (this *IntegrationFixture) LongTestSign3_SES() {
	request := newRequest("GET", "https://email.us-east-1.amazonaws.com/?Action=GetSendStatistics", nil)
	response := sign3AndDo(request)
	this.assertOK(response)
}

func (this *IntegrationFixture) LongTestSign3_Route53() {
	request := newRequest("GET", "https://route53.amazonaws.com/2013-04-01/hostedzone?maxitems=1", nil)
	response := sign3AndDo(request)
	this.assertOK(response)
}

func (this *IntegrationFixture) LongTestSign2_SimpleDB() {
	request := newRequest("GET", "https://sdb.amazonaws.com/?Action=ListDomains&Version=2009-04-15", nil)
	response := sign2AndDo(request)
	this.assertOK(response)
}

func (this *IntegrationFixture) LongTestSignS3Url() {
	s3res := os.Getenv("S3Resource")
	if s3res == "" {
		return
	}
	request, _ := http.NewRequest("GET", s3res, nil)
	response := signS3UrlAndDo(request)
	this.assertOK(response)
}

func TestSign_Version2(t *testing.T) {
	requests := []*http.Request{
		newRequest("GET", "https://ec2.amazonaws.com", url.Values{}),
		newRequest("GET", "https://elasticache.amazonaws.com/", url.Values{}),
	}
	for _, request := range requests {
		signed := Sign(request)
		assertions.New(t).So(signed.URL.Query().Get("SignatureVersion"), should.Equal, "2")
	}
}
func TestSign_Version3(t *testing.T) {
	requests := []*http.Request{
		newRequest("GET", "https://route53.amazonaws.com", url.Values{}),
		newRequest("GET", "https://email.us-east-1.amazonaws.com/", url.Values{}),
	}
	for _, request := range requests {
		signed := Sign(request)
		assertions.New(t).So(signed.Header.Get("X-Amzn-Authorization"), should.NotBeBlank)
	}
}

func TestSign_Version4(t *testing.T) {
	requests := []*http.Request{
		newRequest("POST", "https://sqs.amazonaws.com/", url.Values{}),
		newRequest("GET", "https://iam.amazonaws.com", url.Values{}),
		newRequest("GET", "https://s3.amazonaws.com", url.Values{}),
	}
	for _, request := range requests {
		signed := Sign(request)
		assertions.New(t).So(signed.Header.Get("Authorization"), should.ContainSubstring, ", Signature=")
	}
}

func TestSign_ExistingCredentials_Version2(t *testing.T) {
	requests := []*http.Request{
		newRequest("GET", "https://ec2.amazonaws.com", url.Values{}),
		newRequest("GET", "https://elasticache.amazonaws.com/", url.Values{}),
	}
	for _, request := range requests {
		signed := Sign(request, newKeys())
		assertions.New(t).So(signed.URL.Query().Get("SignatureVersion"), should.Equal, "2")
	}
}

func TestSign_ExistingCredentials_Version3(t *testing.T) {
	requests := []*http.Request{
		newRequest("GET", "https://route53.amazonaws.com", url.Values{}),
		newRequest("GET", "https://email.us-east-1.amazonaws.com/", url.Values{}),
	}
	for _, request := range requests {
		signed := Sign(request, newKeys())
		assertions.New(t).So(signed.Header.Get("X-Amzn-Authorization"), should.NotBeBlank)
	}
}

func TestSign_ExistingCredentials_Version4(t *testing.T) {
	requests := []*http.Request{
		newRequest("POST", "https://sqs.amazonaws.com/", url.Values{}),
		newRequest("GET", "https://iam.amazonaws.com", url.Values{}),
		newRequest("GET", "https://s3.amazonaws.com", url.Values{}),
	}
	for _, request := range requests {
		signed := Sign(request, newKeys())
		assertions.New(t).So(signed.Header.Get("Authorization"), should.ContainSubstring, ", Signature=")
	}
}

func TestExpiration(t *testing.T) {
	assert := assertions.New(t)
	var credentials = &Credentials{}

	// Credentials without an expiration can't expire
	assert.So(credentials.expired(), should.BeFalse)

	// Credentials that expire in 5 minutes aren't expired
	credentials.Expiration = time.Now().Add(5 * time.Minute)
	assert.So(credentials.expired(), should.BeFalse)

	// Credentials that expire in 1 minute are expired
	credentials.Expiration = time.Now().Add(1 * time.Minute)
	assert.So(credentials.expired(), should.BeTrue)

	// Credentials that expired 2 hours ago are expired
	credentials.Expiration = time.Now().Add(-2 * time.Hour)
	assert.So(credentials.expired(), should.BeTrue)
}

func credentialsSet() bool {
	var keys Credentials
	keys = newKeys()
	return keys.AccessKeyID != ""
}

func newRequest(method string, url string, v url.Values) *http.Request {
	request, _ := http.NewRequest(method, url, strings.NewReader(v.Encode()))
	return request
}

func sign2AndDo(request *http.Request) *http.Response {
	Sign2(request)
	response, _ := client.Do(request)
	return response
}

func sign3AndDo(request *http.Request) *http.Response {
	Sign3(request)
	response, _ := client.Do(request)
	return response
}

func sign4AndDo(request *http.Request) *http.Response {
	Sign4(request)
	response, _ := client.Do(request)
	return response
}

func signS3AndDo(request *http.Request) *http.Response {
	SignS3(request)
	response, _ := client.Do(request)
	return response
}

func signS3UrlAndDo(request *http.Request) *http.Response {
	SignS3Url(request, time.Now().AddDate(0, 0, 1))
	response, _ := client.Do(request)
	return response
}

var client = &http.Client{}
