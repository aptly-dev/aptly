package utils

import (
	"io/ioutil"
	
	. "gopkg.in/check.v1"
)

type ChecksumExtraSuite struct{}

var _ = Suite(&ChecksumExtraSuite{})

func (s *ChecksumExtraSuite) TestComplete(c *C) {
	// Test incomplete checksum info
	info := ChecksumInfo{}
	c.Assert(info.Complete(), Equals, false)
	
	// Test with only MD5
	info.MD5 = "d41d8cd98f00b204e9800998ecf8427e"
	c.Assert(info.Complete(), Equals, false)
	
	// Test with MD5 and SHA1
	info.SHA1 = "da39a3ee5e6b4b0d3255bfef95601890afd80709"
	c.Assert(info.Complete(), Equals, false)
	
	// Test with MD5, SHA1, and SHA256
	info.SHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	c.Assert(info.Complete(), Equals, false)
	
	// Test with all checksums present
	info.SHA512 = "cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e"
	c.Assert(info.Complete(), Equals, true)
	
	// Test with empty strings counts as incomplete
	info2 := ChecksumInfo{
		MD5:    "",
		SHA1:   "some",
		SHA256: "some",
		SHA512: "some",
	}
	c.Assert(info2.Complete(), Equals, false)
}

func (s *ChecksumExtraSuite) TestSaveConfigRaw(c *C) {
	tempDir := c.MkDir()
	configFile := tempDir + "/test.conf"
	
	testData := []byte(`{
    "rootDir": "/tmp/aptly",
    "architectures": ["amd64", "i386"],
    "dependencyFollowSuggests": false
}`)
	
	// Test normal save
	err := SaveConfigRaw(configFile, testData)
	c.Assert(err, IsNil)
	
	// Verify file was created with correct content
	data, err := ioutil.ReadFile(configFile)
	c.Assert(err, IsNil)
	c.Assert(data, DeepEquals, testData)
	
	// Test save to invalid path
	err = SaveConfigRaw("/nonexistent/path/config.json", testData)
	c.Assert(err, NotNil)
}

func (s *ChecksumExtraSuite) TestPackagePoolStorageUnmarshalJSON(c *C) {
	// Test unmarshaling Azure type
	azureJSON := `{
		"type": "azure",
		"accountName": "myaccount",
		"accountKey": "mykey",
		"container": "aptly",
		"prefix": "pool"
	}`
	
	var pool PackagePoolStorage
	err := pool.UnmarshalJSON([]byte(azureJSON))
	c.Assert(err, IsNil)
	c.Assert(pool.Azure, NotNil)
	c.Assert(pool.Local, IsNil)
	c.Assert(pool.Azure.AccountName, Equals, "myaccount")
	c.Assert(pool.Azure.Container, Equals, "aptly")
	
	// Test unmarshaling Local type
	localJSON := `{
		"type": "local",
		"path": "/var/aptly/pool"
	}`
	
	var pool2 PackagePoolStorage
	err = pool2.UnmarshalJSON([]byte(localJSON))
	c.Assert(err, IsNil)
	c.Assert(pool2.Local, NotNil)
	c.Assert(pool2.Azure, IsNil)
	c.Assert(pool2.Local.Path, Equals, "/var/aptly/pool")
	
	// Test unmarshaling unknown type
	unknownJSON := `{
		"type": "unknown",
		"some": "data"
	}`
	
	var pool3 PackagePoolStorage
	err = pool3.UnmarshalJSON([]byte(unknownJSON))
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "unknown pool storage type: unknown")
	
	// Test invalid JSON
	var pool4 PackagePoolStorage
	err = pool4.UnmarshalJSON([]byte("invalid json"))
	c.Assert(err, NotNil)
}