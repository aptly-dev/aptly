package utils

import (
	"os"
	"path/filepath"

	. "gopkg.in/check.v1"
)

type ConfigSuite struct {
	config ConfigStructure
}

var _ = Suite(&ConfigSuite{})

func (s *ConfigSuite) TestLoadConfig(c *C) {
	configname := filepath.Join(c.MkDir(), "aptly.json")
	f, _ := os.Create(configname)
	f.WriteString(configFile)
	f.Close()

	err := LoadConfig(configname, &s.config)
	c.Assert(err, IsNil)
	c.Check(s.config.RootDir, Equals, "/opt/aptly/")
	c.Check(s.config.DownloadConcurrency, Equals, 33)
	c.Check(s.config.DatabaseOpenAttempts, Equals, 33)
}

func (s *ConfigSuite) TestSaveConfig(c *C) {
	configname := filepath.Join(c.MkDir(), "aptly.json")

	s.config.RootDir = "/tmp/aptly"
	s.config.DownloadConcurrency = 5
	s.config.DatabaseOpenAttempts = 5
	s.config.GpgProvider = "gpg"

	s.config.FileSystemPublishRoots = map[string]FileSystemPublishRoot{"test": {
		RootDir: "/opt/aptly-publish"}}

	s.config.S3PublishRoots = map[string]S3PublishRoot{"test": {
		Region: "us-east-1",
		Bucket: "repo"}}

	s.config.SwiftPublishRoots = map[string]SwiftPublishRoot{"test": {
		Container: "repo"}}

	err := SaveConfig(configname, &s.config)
	c.Assert(err, IsNil)

	f, _ := os.Open(configname)
	defer f.Close()

	st, _ := f.Stat()
	buf := make([]byte, st.Size())
	f.Read(buf)

	c.Check(string(buf), Equals, ""+
		"{\n"+
		"  \"rootDir\": \"/tmp/aptly\",\n"+
		"  \"downloadConcurrency\": 5,\n"+
		"  \"downloadSpeedLimit\": 0,\n"+
		"  \"downloadRetries\": 0,\n"+
		"  \"databaseOpenAttempts\": 5,\n"+
		"  \"architectures\": null,\n"+
		"  \"dependencyFollowSuggests\": false,\n"+
		"  \"dependencyFollowRecommends\": false,\n"+
		"  \"dependencyFollowAllVariants\": false,\n"+
		"  \"dependencyFollowSource\": false,\n"+
		"  \"dependencyVerboseResolve\": false,\n"+
		"  \"gpgDisableSign\": false,\n"+
		"  \"gpgDisableVerify\": false,\n"+
		"  \"gpgProvider\": \"gpg\",\n"+
		"  \"downloadSourcePackages\": false,\n"+
		"  \"skipLegacyPool\": false,\n"+
		"  \"ppaDistributorID\": \"\",\n"+
		"  \"ppaCodename\": \"\",\n"+
		"  \"skipContentsPublishing\": false,\n"+
		"  \"FileSystemPublishEndpoints\": {\n"+
		"    \"test\": {\n"+
		"      \"rootDir\": \"/opt/aptly-publish\",\n"+
		"      \"linkMethod\": \"\",\n"+
		"      \"verifyMethod\": \"\"\n"+
		"    }\n"+
		"  },\n"+
		"  \"S3PublishEndpoints\": {\n"+
		"    \"test\": {\n"+
		"      \"region\": \"us-east-1\",\n"+
		"      \"bucket\": \"repo\",\n"+
		"      \"endpoint\": \"\",\n"+
		"      \"awsAccessKeyID\": \"\",\n"+
		"      \"awsSecretAccessKey\": \"\",\n"+
		"      \"awsSessionToken\": \"\",\n"+
		"      \"prefix\": \"\",\n"+
		"      \"acl\": \"\",\n"+
		"      \"storageClass\": \"\",\n"+
		"      \"encryptionMethod\": \"\",\n"+
		"      \"plusWorkaround\": false,\n"+
		"      \"disableMultiDel\": false,\n"+
		"      \"forceSigV2\": false,\n"+
		"      \"debug\": false\n"+
		"    }\n"+
		"  },\n"+
		"  \"SwiftPublishEndpoints\": {\n"+
		"    \"test\": {\n"+
		"      \"osname\": \"\",\n"+
		"      \"password\": \"\",\n"+
		"      \"authurl\": \"\",\n"+
		"      \"tenant\": \"\",\n"+
		"      \"tenantid\": \"\",\n"+
		"      \"domain\": \"\",\n"+
		"      \"domainid\": \"\",\n"+
		"      \"tenantdomain\": \"\",\n"+
		"      \"tenantdomainid\": \"\",\n"+
		"      \"prefix\": \"\",\n"+
		"      \"container\": \"repo\"\n"+
		"    }\n"+
		"  },\n"+
		"  \"AsyncAPI\": false\n"+
		"}")
}

const configFile = `{"rootDir": "/opt/aptly/", "downloadConcurrency": 33, "databaseOpenAttempts": 33}`
