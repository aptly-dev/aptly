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
	c.Check(s.config.GetRootDir(), Equals, "/opt/aptly/")
	c.Check(s.config.DownloadConcurrency, Equals, 33)
	c.Check(s.config.DatabaseOpenAttempts, Equals, 33)
}

func (s *ConfigSuite) TestSaveConfig(c *C) {
	configname := filepath.Join(c.MkDir(), "aptly.json")

	s.config.RootDir = "/tmp/aptly"
	s.config.DownloadConcurrency = 5
	s.config.DatabaseOpenAttempts = 5
	s.config.GpgProvider = "gpg"

	s.config.PackagePoolStorage.Local = &LocalPoolStorage{"/tmp/aptly-pool"}

	s.config.FileSystemPublishRoots = map[string]FileSystemPublishRoot{"test": {
		RootDir: "/opt/aptly-publish"}}

	s.config.S3PublishRoots = map[string]S3PublishRoot{"test": {
		Region: "us-east-1",
		Bucket: "repo"}}

	s.config.SwiftPublishRoots = map[string]SwiftPublishRoot{"test": {
		Container: "repo"}}

	s.config.AzurePublishRoots = map[string]AzureEndpoint{"test": {
		Container: "repo"}}

	s.config.LogLevel = "info"
	s.config.LogFormat = "json"

	err := SaveConfig(configname, &s.config)
	c.Assert(err, IsNil)

	f, _ := os.Open(configname)
	defer f.Close()

	st, _ := f.Stat()
	buf := make([]byte, st.Size())
	f.Read(buf)

	c.Check(string(buf), Equals, ""+
		"{\n" +
		"  \"rootDir\": \"/tmp/aptly\",\n" +
		"  \"logLevel\": \"info\",\n" +
		"  \"logFormat\": \"json\",\n" +
		"  \"databaseOpenAttempts\": 5,\n" +
		"  \"architectures\": null,\n" +
		"  \"skipLegacyPool\": false,\n" +
		"  \"dependencyFollowSuggests\": false,\n" +
		"  \"dependencyFollowRecommends\": false,\n" +
		"  \"dependencyFollowAllVariants\": false,\n" +
		"  \"dependencyFollowSource\": false,\n" +
		"  \"dependencyVerboseResolve\": false,\n" +
		"  \"ppaDistributorID\": \"\",\n" +
		"  \"ppaCodename\": \"\",\n" +
		"  \"serveInAPIMode\": false,\n" +
		"  \"enableMetricsEndpoint\": false,\n" +
		"  \"enableSwaggerEndpoint\": false,\n" +
		"  \"AsyncAPI\": false,\n" +
		"  \"databaseBackend\": {\n" +
		"    \"type\": \"\",\n" +
		"    \"dbPath\": \"\",\n" +
		"    \"url\": \"\"\n" +
		"  },\n" +
		"  \"downloader\": \"\",\n" +
		"  \"downloadConcurrency\": 5,\n" +
		"  \"downloadSpeedLimit\": 0,\n" +
		"  \"downloadRetries\": 0,\n" +
		"  \"downloadSourcePackages\": false,\n" +
		"  \"gpgProvider\": \"gpg\",\n" +
		"  \"gpgDisableSign\": false,\n" +
		"  \"gpgDisableVerify\": false,\n" +
		"  \"skipContentsPublishing\": false,\n" +
		"  \"skipBz2Publishing\": false,\n" +
		"  \"FileSystemPublishEndpoints\": {\n" +
		"    \"test\": {\n" +
		"      \"rootDir\": \"/opt/aptly-publish\",\n" +
		"      \"linkMethod\": \"\",\n" +
		"      \"verifyMethod\": \"\"\n" +
		"    }\n" +
		"  },\n" +
		"  \"S3PublishEndpoints\": {\n" +
		"    \"test\": {\n" +
		"      \"region\": \"us-east-1\",\n" +
		"      \"bucket\": \"repo\",\n" +
		"      \"prefix\": \"\",\n" +
		"      \"acl\": \"\",\n" +
		"      \"awsAccessKeyID\": \"\",\n" +
		"      \"awsSecretAccessKey\": \"\",\n" +
		"      \"awsSessionToken\": \"\",\n" +
		"      \"endpoint\": \"\",\n" +
		"      \"storageClass\": \"\",\n" +
		"      \"encryptionMethod\": \"\",\n" +
		"      \"plusWorkaround\": false,\n" +
		"      \"disableMultiDel\": false,\n" +
		"      \"forceSigV2\": false,\n" +
		"      \"forceVirtualHostedStyle\": false,\n" +
		"      \"debug\": false\n" +
		"    }\n" +
		"  },\n" +
		"  \"SwiftPublishEndpoints\": {\n" +
		"    \"test\": {\n" +
		"      \"container\": \"repo\",\n" +
		"      \"prefix\": \"\",\n" +
		"      \"osname\": \"\",\n" +
		"      \"password\": \"\",\n" +
		"      \"tenant\": \"\",\n" +
		"      \"tenantid\": \"\",\n" +
		"      \"domain\": \"\",\n" +
		"      \"domainid\": \"\",\n" +
		"      \"tenantdomain\": \"\",\n" +
		"      \"tenantdomainid\": \"\",\n" +
		"      \"authurl\": \"\"\n" +
		"    }\n" +
		"  },\n" +
		"  \"AzurePublishEndpoints\": {\n" +
		"    \"test\": {\n" +
		"      \"container\": \"repo\",\n" +
		"      \"prefix\": \"\",\n" +
		"      \"accountName\": \"\",\n" +
		"      \"accountKey\": \"\",\n" +
		"      \"endpoint\": \"\"\n" +
		"    }\n" +
		"  },\n" +
		"  \"packagePoolStorage\": {\n" +
		"    \"type\": \"local\",\n" +
		"    \"path\": \"/tmp/aptly-pool\"\n" +
		"  }\n" +
		"}")
}

const configFile = `{"rootDir": "/opt/aptly/", "downloadConcurrency": 33, "databaseOpenAttempts": 33}`
