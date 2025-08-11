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
	configname := filepath.Join(c.MkDir(), "aptly.json1")
	f, _ := os.Create(configname)
	_, _ = f.WriteString(configFile)
	_ = f.Close()

	// start with empty config
	s.config = ConfigStructure{}

	err := LoadConfig(configname, &s.config)
	c.Assert(err, IsNil)
	c.Check(s.config.GetRootDir(), Equals, "/opt/aptly/")
	c.Check(s.config.DownloadConcurrency, Equals, 33)
	c.Check(s.config.DatabaseOpenAttempts, Equals, 33)
}

func (s *ConfigSuite) TestSaveConfig(c *C) {
	configname := filepath.Join(c.MkDir(), "aptly.json2")

	// start with empty config
	s.config = ConfigStructure{}

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
	defer func() {
		_ = f.Close()
	}()

	st, _ := f.Stat()
	buf := make([]byte, st.Size())
	_, _ = f.Read(buf)

	c.Check(string(buf), Equals, ""+
		"{\n"+
		"  \"rootDir\": \"/tmp/aptly\",\n"+
		"  \"logLevel\": \"info\",\n"+
		"  \"logFormat\": \"json\",\n"+
		"  \"databaseOpenAttempts\": 5,\n"+
		"  \"architectures\": null,\n"+
		"  \"skipLegacyPool\": false,\n"+
		"  \"dependencyFollowSuggests\": false,\n"+
		"  \"dependencyFollowRecommends\": false,\n"+
		"  \"dependencyFollowAllVariants\": false,\n"+
		"  \"dependencyFollowSource\": false,\n"+
		"  \"dependencyVerboseResolve\": false,\n"+
		"  \"ppaDistributorID\": \"\",\n"+
		"  \"ppaCodename\": \"\",\n"+
		"  \"serveInAPIMode\": false,\n"+
		"  \"enableMetricsEndpoint\": false,\n"+
		"  \"enableSwaggerEndpoint\": false,\n"+
		"  \"AsyncAPI\": false,\n"+
		"  \"databaseBackend\": {\n"+
		"    \"type\": \"\",\n"+
		"    \"dbPath\": \"\",\n"+
		"    \"url\": \"\"\n"+
		"  },\n"+
		"  \"downloader\": \"\",\n"+
		"  \"downloadConcurrency\": 5,\n"+
		"  \"downloadSpeedLimit\": 0,\n"+
		"  \"downloadRetries\": 0,\n"+
		"  \"downloadSourcePackages\": false,\n"+
		"  \"gpgProvider\": \"gpg\",\n"+
		"  \"gpgDisableSign\": false,\n"+
		"  \"gpgDisableVerify\": false,\n"+
		"  \"gpgKeys\": null,\n"+
		"  \"skipContentsPublishing\": false,\n"+
		"  \"skipBz2Publishing\": false,\n"+
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
		"      \"prefix\": \"\",\n"+
		"      \"acl\": \"\",\n"+
		"      \"awsAccessKeyID\": \"\",\n"+
		"      \"awsSecretAccessKey\": \"\",\n"+
		"      \"awsSessionToken\": \"\",\n"+
		"      \"endpoint\": \"\",\n"+
		"      \"storageClass\": \"\",\n"+
		"      \"encryptionMethod\": \"\",\n"+
		"      \"plusWorkaround\": false,\n"+
		"      \"disableMultiDel\": false,\n"+
		"      \"forceSigV2\": false,\n"+
		"      \"forceVirtualHostedStyle\": false,\n"+
		"      \"debug\": false\n"+
		"    }\n"+
		"  },\n"+
		"  \"SwiftPublishEndpoints\": {\n"+
		"    \"test\": {\n"+
		"      \"container\": \"repo\",\n"+
		"      \"prefix\": \"\",\n"+
		"      \"osname\": \"\",\n"+
		"      \"password\": \"\",\n"+
		"      \"tenant\": \"\",\n"+
		"      \"tenantid\": \"\",\n"+
		"      \"domain\": \"\",\n"+
		"      \"domainid\": \"\",\n"+
		"      \"tenantdomain\": \"\",\n"+
		"      \"tenantdomainid\": \"\",\n"+
		"      \"authurl\": \"\"\n"+
		"    }\n"+
		"  },\n"+
		"  \"AzurePublishEndpoints\": {\n"+
		"    \"test\": {\n"+
		"      \"container\": \"repo\",\n"+
		"      \"prefix\": \"\",\n"+
		"      \"accountName\": \"\",\n"+
		"      \"accountKey\": \"\",\n"+
		"      \"endpoint\": \"\"\n"+
		"    }\n"+
		"  },\n"+
		"  \"packagePoolStorage\": {\n"+
		"    \"type\": \"local\",\n"+
		"    \"path\": \"/tmp/aptly-pool\"\n"+
		"  }\n"+
		"}")
}

func (s *ConfigSuite) TestLoadYAMLConfig(c *C) {
	configname := filepath.Join(c.MkDir(), "aptly.yaml1")
	f, _ := os.Create(configname)
	_, _ = f.WriteString(configFileYAML)
	_ = f.Close()

	// start with empty config
	s.config = ConfigStructure{}

	err := LoadConfig(configname, &s.config)
	c.Assert(err, IsNil)
	c.Check(s.config.GetRootDir(), Equals, "/opt/aptly/")
	c.Check(s.config.DownloadConcurrency, Equals, 40)
	c.Check(s.config.DatabaseOpenAttempts, Equals, 10)
}

func (s *ConfigSuite) TestLoadYAMLErrorConfig(c *C) {
	configname := filepath.Join(c.MkDir(), "aptly.yaml2")
	f, _ := os.Create(configname)
	_, _ = f.WriteString(configFileYAMLError)
	_ = f.Close()

	// start with empty config
	s.config = ConfigStructure{}

	err := LoadConfig(configname, &s.config)
	c.Assert(err.Error(), Equals, "invalid yaml (unknown pool storage type: invalid) or json (invalid character 'p' looking for beginning of value)")
}

func (s *ConfigSuite) TestSaveYAMLConfig(c *C) {
	configname := filepath.Join(c.MkDir(), "aptly.yaml3")
	f, _ := os.Create(configname)
	_, _ = f.WriteString(configFileYAML)
	_ = f.Close()

	// start with empty config
	s.config = ConfigStructure{}

	err := LoadConfig(configname, &s.config)
	c.Assert(err, IsNil)

	err = SaveConfigYAML(configname, &s.config)
	c.Assert(err, IsNil)

	f, _ = os.Open(configname)
	defer func() {
		_ = f.Close()
	}()

	st, _ := f.Stat()
	buf := make([]byte, st.Size())
	_, _ = f.Read(buf)

	c.Check(string(buf), Equals, configFileYAML)
}

func (s *ConfigSuite) TestSaveYAML2Config(c *C) {
	// start with empty config
	s.config = ConfigStructure{}

	s.config.PackagePoolStorage.Local = &LocalPoolStorage{"/tmp/aptly-pool"}
	s.config.PackagePoolStorage.Azure = nil

	configname := filepath.Join(c.MkDir(), "aptly.yaml4")
	err := SaveConfigYAML(configname, &s.config)
	c.Assert(err, IsNil)

	f, _ := os.Open(configname)
	defer func() {
		_ = f.Close()
	}()

	st, _ := f.Stat()
	buf := make([]byte, st.Size())
	_, _ = f.Read(buf)

	c.Check(string(buf), Equals, ""+
		"root_dir: \"\"\n"+
		"log_level: \"\"\n"+
		"log_format: \"\"\n"+
		"database_open_attempts: 0\n"+
		"architectures: []\n"+
		"skip_legacy_pool: false\n"+
		"dep_follow_suggests: false\n"+
		"dep_follow_recommends: false\n"+
		"dep_follow_all_variants: false\n"+
		"dep_follow_source: false\n"+
		"dep_verboseresolve: false\n"+
		"ppa_distributor_id: \"\"\n"+
		"ppa_codename: \"\"\n"+
		"serve_in_api_mode: false\n"+
		"enable_metrics_endpoint: false\n"+
		"enable_swagger_endpoint: false\n"+
		"async_api: false\n"+
		"database_backend:\n"+
		"    type: \"\"\n"+
		"    db_path: \"\"\n"+
		"    url: \"\"\n"+
		"downloader: \"\"\n"+
		"download_concurrency: 0\n"+
		"download_limit: 0\n"+
		"download_retries: 0\n"+
		"download_sourcepackages: false\n"+
		"gpg_provider: \"\"\n"+
		"gpg_disable_sign: false\n"+
		"gpg_disable_verify: false\n"+
		"gpg_keys: []\n"+
		"skip_contents_publishing: false\n"+
		"skip_bz2_publishing: false\n"+
		"filesystem_publish_endpoints: {}\n"+
		"s3_publish_endpoints: {}\n"+
		"swift_publish_endpoints: {}\n"+
		"azure_publish_endpoints: {}\n"+
		"packagepool_storage:\n"+
		"    type: local\n"+
		"    path: /tmp/aptly-pool\n")
}

func (s *ConfigSuite) TestLoadEmptyConfig(c *C) {
	configname := filepath.Join(c.MkDir(), "aptly.yaml5")
	f, _ := os.Create(configname)
	_ = f.Close()

	// start with empty config
	s.config = ConfigStructure{}

	err := LoadConfig(configname, &s.config)
	c.Assert(err.Error(), Equals, "invalid yaml (EOF) or json (EOF)")
}

const configFile = `{"rootDir": "/opt/aptly/", "downloadConcurrency": 33, "databaseOpenAttempts": 33}`
const configFileYAML = `root_dir: /opt/aptly/
log_level: error
log_format: json
database_open_attempts: 10
architectures:
    - amd64
    - arm64
skip_legacy_pool: true
dep_follow_suggests: true
dep_follow_recommends: true
dep_follow_all_variants: true
dep_follow_source: true
dep_verboseresolve: true
ppa_distributor_id: Ubuntu
ppa_codename: code
serve_in_api_mode: true
enable_metrics_endpoint: true
enable_swagger_endpoint: true
async_api: true
database_backend:
    type: etcd
    db_path: ""
    url: 127.0.0.1:2379
downloader: grab
download_concurrency: 40
download_limit: 100
download_retries: 10
download_sourcepackages: true
gpg_provider: gpg
gpg_disable_sign: true
gpg_disable_verify: true
gpg_keys: []
skip_contents_publishing: true
skip_bz2_publishing: true
filesystem_publish_endpoints:
    test1:
        root_dir: /opt/srv/aptly_public
        link_method: hardlink
        verify_method: md5
s3_publish_endpoints:
    test:
        region: us-east-1
        bucket: test-bucket
        prefix: prfx
        acl: public-read
        access_key_id: "2"
        secret_access_key: secret
        session_token: none
        endpoint: endpoint
        storage_class: STANDARD
        encryption_method: AES256
        plus_workaround: true
        disable_multidel: true
        force_sigv2: true
        force_virtualhosted_style: true
        debug: true
swift_publish_endpoints:
    test:
        container: c1
        prefix: pre
        username: user
        password: pass
        tenant: t
        tenant_id: "2"
        domain: pop
        domain_id: "1"
        tenant_domain: td
        tenant_domain_id: "3"
        auth_url: http://auth.url
azure_publish_endpoints:
    test:
        container: container1
        prefix: pre2
        account_name: aname
        account_key: akey
        endpoint: https://end.point
packagepool_storage:
    type: azure
    container: test-pool1
    prefix: pre3
    account_name: a name
    account_key: a key
    endpoint: ep
`
const configFileYAMLError = `packagepool_storage:
    type: invalid
`
