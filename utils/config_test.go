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

	s.config.JFrogPublishRoots = map[string]JFrogPublishRoot{"test": {
		Repository: "repo",
		Url:        "jfrog.example.com"}}

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

	// FIXME: "  \"ppaBaseURL\": \"\",\n" +
	c.Check(string(buf), Equals, ""+
"root_dir: \"\"
" +
    "log_level: \"\"
" +
    "log_format: \"\"
" +
    "database_open_attempts: 0
" +
    "architectures: []
" +
    "skip_legacy_pool: false
" +
    "dep_follow_suggests: false
" +
    "dep_follow_recommends: false
" +
    "dep_follow_all_variants: false
" +
    "dep_follow_source: false
" +
    "dep_verboseresolve: false
" +
    "ppa_distributor_id: \"\"
" +
    "ppa_codename: \"\"
" +
    "ppa_baseurl: \"\"
" +
    "serve_in_api_mode: false
" +
    "enable_metrics_endpoint: false
" +
    "enable_swagger_endpoint: false
" +
    "async_api: false
" +
    "database_backend:
" +
    "    type: \"\"
" +
    "    db_path: \"\"
" +
    "    url: \"\"
" +
    "downloader: \"\"
" +
    "download_concurrency: 0
" +
    "download_limit: 0
" +
    "download_retries: 0
" +
    "download_sourcepackages: false
" +
    "gpg_provider: \"\"
" +
    "gpg_disable_sign: false
" +
    "gpg_disable_verify: false
" +
    "gpg_keys: []
" +
    "skip_contents_publishing: false
" +
    "skip_bz2_publishing: false
" +
    "filesystem_publish_endpoints: {}
" +
    "jfrog_publish_endpoints: {}
" +
    "s3_publish_endpoints: {}
" +
    "swift_publish_endpoints: {}
" +
    "azure_publish_endpoints: {}
" +
    "packagepool_storage:
" +
    "    type: local
" +
    "    path: /tmp/aptly-pool
")
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
ppa_baseurl: http://ppa.launchpad.net
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
