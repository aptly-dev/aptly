package utils

import (
	. "gopkg.in/check.v1"
)

type ConfigAccessorSuite struct{}

var _ = Suite(&ConfigAccessorSuite{})

func (s *ConfigAccessorSuite) TestGetFileSystemPublishRoots(c *C) {
	// Test with empty map
	config := ConfigStructure{
		FileSystemPublishRoots: map[string]FileSystemPublishRoot{},
	}
	roots := config.GetFileSystemPublishRoots()
	c.Assert(roots, NotNil)
	c.Assert(len(roots), Equals, 0)

	// Test with populated map
	config.FileSystemPublishRoots = map[string]FileSystemPublishRoot{
		"test1": {RootDir: "/tmp/test1", LinkMethod: "hardlink"},
		"test2": {RootDir: "/tmp/test2", LinkMethod: "copy", VerifyMethod: "md5"},
	}

	roots = config.GetFileSystemPublishRoots()
	c.Assert(len(roots), Equals, 2)
	c.Assert(roots["test1"].RootDir, Equals, "/tmp/test1")
	c.Assert(roots["test1"].LinkMethod, Equals, "hardlink")
	c.Assert(roots["test2"].RootDir, Equals, "/tmp/test2")
	c.Assert(roots["test2"].LinkMethod, Equals, "copy")
	c.Assert(roots["test2"].VerifyMethod, Equals, "md5")

	// Verify it's a copy by modifying the returned map
	roots["test3"] = FileSystemPublishRoot{RootDir: "/tmp/test3"}
	c.Assert(len(config.FileSystemPublishRoots), Equals, 2) // Original unchanged
}

func (s *ConfigAccessorSuite) TestGetS3PublishRoots(c *C) {
	// Test with empty map
	config := ConfigStructure{
		S3PublishRoots: map[string]S3PublishRoot{},
	}
	roots := config.GetS3PublishRoots()
	c.Assert(roots, NotNil)
	c.Assert(len(roots), Equals, 0)

	// Test with populated map
	config.S3PublishRoots = map[string]S3PublishRoot{
		"bucket1": {
			Region:      "us-east-1",
			Bucket:      "my-bucket-1",
			AccessKeyID: "key1",
			ACL:         "public-read",
		},
		"bucket2": {
			Region: "eu-west-1",
			Bucket: "my-bucket-2",
			Prefix: "aptly",
			ACL:    "private",
		},
	}

	roots = config.GetS3PublishRoots()
	c.Assert(len(roots), Equals, 2)
	c.Assert(roots["bucket1"].Region, Equals, "us-east-1")
	c.Assert(roots["bucket1"].Bucket, Equals, "my-bucket-1")
	c.Assert(roots["bucket2"].Prefix, Equals, "aptly")

	// Verify it's a copy
	roots["bucket3"] = S3PublishRoot{Bucket: "new-bucket"}
	c.Assert(len(config.S3PublishRoots), Equals, 2) // Original unchanged
}

func (s *ConfigAccessorSuite) TestGetSwiftPublishRoots(c *C) {
	// Test with empty map
	config := ConfigStructure{
		SwiftPublishRoots: map[string]SwiftPublishRoot{},
	}
	roots := config.GetSwiftPublishRoots()
	c.Assert(roots, NotNil)
	c.Assert(len(roots), Equals, 0)

	// Test with populated map
	config.SwiftPublishRoots = map[string]SwiftPublishRoot{
		"container1": {
			Container: "aptly-container",
			Prefix:    "debian",
			UserName:  "user1",
			AuthURL:   "https://auth.example.com",
		},
	}

	roots = config.GetSwiftPublishRoots()
	c.Assert(len(roots), Equals, 1)
	c.Assert(roots["container1"].Container, Equals, "aptly-container")
	c.Assert(roots["container1"].Prefix, Equals, "debian")

	// Verify it's a copy
	delete(roots, "container1")
	c.Assert(len(config.SwiftPublishRoots), Equals, 1) // Original unchanged
}

func (s *ConfigAccessorSuite) TestGetAzurePublishRoots(c *C) {
	// Test with empty map
	config := ConfigStructure{
		AzurePublishRoots: map[string]AzureEndpoint{},
	}
	roots := config.GetAzurePublishRoots()
	c.Assert(roots, NotNil)
	c.Assert(len(roots), Equals, 0)

	// Test with populated map
	config.AzurePublishRoots = map[string]AzureEndpoint{
		"storage1": {
			AccountName: "myaccount",
			AccountKey:  "mykey",
			Container:   "aptly",
			Prefix:      "repos",
			Endpoint:    "https://myaccount.blob.core.windows.net",
		},
		"storage2": {
			AccountName: "account2",
			Container:   "debian",
		},
	}

	roots = config.GetAzurePublishRoots()
	c.Assert(len(roots), Equals, 2)
	c.Assert(roots["storage1"].AccountName, Equals, "myaccount")
	c.Assert(roots["storage1"].Container, Equals, "aptly")
	c.Assert(roots["storage2"].Container, Equals, "debian")

	// Verify it's a copy
	modified := roots["storage1"]
	modified.Container = "modified"
	roots["storage1"] = modified
	c.Assert(config.AzurePublishRoots["storage1"].Container, Equals, "aptly") // Original unchanged
}
