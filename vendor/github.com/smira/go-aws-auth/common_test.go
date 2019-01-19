package awsauth

import (
	"net/url"
	"testing"

	"github.com/smartystreets/assertions/should"
	"github.com/smartystreets/gunit"
)

func TestCommonFixture(t *testing.T) {
	gunit.Run(new(CommonFixture), t)
}

type CommonFixture struct {
	*gunit.Fixture
}

func (this *CommonFixture) serviceAndRegion(id string) []string {
	service, region := serviceAndRegion(id)
	return []string{service, region}
}
func (this *CommonFixture) TestServiceAndRegion() {
	this.So(this.serviceAndRegion("sqs.us-west-2.amazonaws.com"), should.Resemble, []string{"sqs", "us-west-2"})
	this.So(this.serviceAndRegion("iam.amazonaws.com"), should.Resemble, []string{"iam", "us-east-1"})
	this.So(this.serviceAndRegion("sns.us-west-2.amazonaws.com"), should.Resemble, []string{"sns", "us-west-2"})
	this.So(this.serviceAndRegion("bucketname.s3.amazonaws.com"), should.Resemble, []string{"s3", "us-east-1"})
	this.So(this.serviceAndRegion("s3.amazonaws.com"), should.Resemble, []string{"s3", "us-east-1"})
	this.So(this.serviceAndRegion("s3-us-west-1.amazonaws.com"), should.Resemble, []string{"s3", "us-west-1"})
	this.So(this.serviceAndRegion("s3-external-1.amazonaws.com"), should.Resemble, []string{"s3", "us-east-1"})
}

func (this *CommonFixture) TestHashFunctions() {
	this.So(hashMD5([]byte("Pretend this is a REALLY long byte array...")), should.Equal, "KbVTY8Vl6VccnzQf1AGOFw==")
	this.So(hashSHA256([]byte("This is... Sparta!!")), should.Equal,
		"5c81a4ef1172e89b1a9d575f4cd82f4ed20ea9137e61aa7f1ab936291d24e79a")

	key := []byte("asdf1234")
	contents := "SmartyStreets was here"

	expectedHMAC_SHA256 := []byte{
		65, 46, 186, 78, 2, 155, 71, 104, 49, 37, 5, 66, 195, 129, 159, 227,
		239, 53, 240, 107, 83, 21, 235, 198, 238, 216, 108, 149, 143, 222, 144, 94}
	this.So(hmacSHA256(key, contents), should.Resemble, expectedHMAC_SHA256)

	expectedHMAC_SHA1 := []byte{
		164, 77, 252, 0, 87, 109, 207, 110, 163, 75, 228, 122, 83, 255, 233, 237, 125, 206, 85, 70}
	this.So(hmacSHA1(key, contents), should.Resemble, expectedHMAC_SHA1)
}

func (this *CommonFixture) TestConcat() {
	this.So(concat("\n", "Test1", "Test2"), should.Equal, "Test1\nTest2")
	this.So(concat(".", "Test1"), should.Equal, "Test1")
	this.So(concat("\t", "1", "2", "3", "4"), should.Equal, "1\t2\t3\t4")
}

func (this *CommonFixture) TestURINormalization() {
	this.So(
		normuri("/-._~0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"), should.Equal,
		"/-._~0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")

	this.So(normuri("/ /foo"), should.Equal, "/%20/foo")
	this.So(normuri("/(foo)"), should.Equal, "/%28foo%29")

	this.So(
		normquery(url.Values{"p": []string{" +&;-=._~0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"}}),
		should.Equal,
		"p=%20%2B%26%3B-%3D._~0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")
}
