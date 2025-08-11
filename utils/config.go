package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DisposaBoy/JsonConfigReader"
	yaml "gopkg.in/yaml.v3"
)

// ConfigStructure is structure of main configuration
type ConfigStructure struct { // nolint: maligned
	// General
	RootDir              string   `json:"rootDir"                       yaml:"root_dir"`
	LogLevel             string   `json:"logLevel"                      yaml:"log_level"`
	LogFormat            string   `json:"logFormat"                     yaml:"log_format"`
	DatabaseOpenAttempts int      `json:"databaseOpenAttempts"          yaml:"database_open_attempts"`
	Architectures        []string `json:"architectures"                 yaml:"architectures"`
	SkipLegacyPool       bool     `json:"skipLegacyPool"                yaml:"skip_legacy_pool"` // OBSOLETE

	// Dependency following
	DepFollowSuggests    bool `json:"dependencyFollowSuggests"      yaml:"dep_follow_suggests"`
	DepFollowRecommends  bool `json:"dependencyFollowRecommends"    yaml:"dep_follow_recommends"`
	DepFollowAllVariants bool `json:"dependencyFollowAllVariants"   yaml:"dep_follow_all_variants"`
	DepFollowSource      bool `json:"dependencyFollowSource"        yaml:"dep_follow_source"`
	DepVerboseResolve    bool `json:"dependencyVerboseResolve"      yaml:"dep_verboseresolve"`

	// PPA
	PpaDistributorID string `json:"ppaDistributorID"              yaml:"ppa_distributor_id"`
	PpaCodename      string `json:"ppaCodename"                   yaml:"ppa_codename"`

	// Server
	ServeInAPIMode        bool `json:"serveInAPIMode"                yaml:"serve_in_api_mode"`
	EnableMetricsEndpoint bool `json:"enableMetricsEndpoint"         yaml:"enable_metrics_endpoint"`
	EnableSwaggerEndpoint bool `json:"enableSwaggerEndpoint"         yaml:"enable_swagger_endpoint"`
	AsyncAPI              bool `json:"AsyncAPI"                      yaml:"async_api"` // OBSOLETE

	// Database
	DatabaseBackend DBConfig `json:"databaseBackend"               yaml:"database_backend"`

	// Mirroring
	Downloader             string `json:"downloader"                    yaml:"downloader"`
	DownloadConcurrency    int    `json:"downloadConcurrency"           yaml:"download_concurrency"`
	DownloadLimit          int64  `json:"downloadSpeedLimit"            yaml:"download_limit"`
	DownloadRetries        int    `json:"downloadRetries"               yaml:"download_retries"`
	DownloadSourcePackages bool   `json:"downloadSourcePackages"        yaml:"download_sourcepackages"`

	// Signing
	GpgProvider      string   `json:"gpgProvider"                   yaml:"gpg_provider"`
	GpgDisableSign   bool     `json:"gpgDisableSign"                yaml:"gpg_disable_sign"`
	GpgDisableVerify bool     `json:"gpgDisableVerify"              yaml:"gpg_disable_verify"`
	GpgKeys          []string `json:"gpgKeys"                       yaml:"gpg_keys"`

	// Publishing
	SkipContentsPublishing bool `json:"skipContentsPublishing"        yaml:"skip_contents_publishing"`
	SkipBz2Publishing      bool `json:"skipBz2Publishing"             yaml:"skip_bz2_publishing"`

	// Storage
	FileSystemPublishRoots map[string]FileSystemPublishRoot `json:"FileSystemPublishEndpoints"    yaml:"filesystem_publish_endpoints"`
	S3PublishRoots         map[string]S3PublishRoot         `json:"S3PublishEndpoints"            yaml:"s3_publish_endpoints"`
	SwiftPublishRoots      map[string]SwiftPublishRoot      `json:"SwiftPublishEndpoints"         yaml:"swift_publish_endpoints"`
	AzurePublishRoots      map[string]AzureEndpoint         `json:"AzurePublishEndpoints"         yaml:"azure_publish_endpoints"`
	PackagePoolStorage     PackagePoolStorage               `json:"packagePoolStorage"            yaml:"packagepool_storage"`
}

// DBConfig structure
type DBConfig struct {
	Type   string `json:"type"    yaml:"type"`
	DBPath string `json:"dbPath"  yaml:"db_path"`
	URL    string `json:"url"     yaml:"url"`
}

type LocalPoolStorage struct {
	Path string `json:"path,omitempty"  yaml:"path,omitempty"`
}

type PackagePoolStorage struct {
	Local *LocalPoolStorage
	Azure *AzureEndpoint
}

var AZURE = "azure"
var LOCAL = "local"

func (pool *PackagePoolStorage) UnmarshalJSON(data []byte) error {
	var discriminator struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(data, &discriminator); err != nil {
		return err
	}

	switch discriminator.Type {
	case AZURE:
		pool.Azure = &AzureEndpoint{}
		return json.Unmarshal(data, &pool.Azure)
	case LOCAL, "":
		pool.Local = &LocalPoolStorage{}
		return json.Unmarshal(data, &pool.Local)
	default:
		return fmt.Errorf("unknown pool storage type: %s", discriminator.Type)
	}
}

func (pool *PackagePoolStorage) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var discriminator struct {
		Type string `yaml:"type"`
	}
	if err := unmarshal(&discriminator); err != nil {
		return err
	}

	switch discriminator.Type {
	case AZURE:
		pool.Azure = &AzureEndpoint{}
		return unmarshal(&pool.Azure)
	case LOCAL, "":
		pool.Local = &LocalPoolStorage{}
		return unmarshal(&pool.Local)
	default:
		return fmt.Errorf("unknown pool storage type: %s", discriminator.Type)
	}
}

func (pool *PackagePoolStorage) MarshalJSON() ([]byte, error) {
	var wrapper struct {
		Type string `json:"type,omitempty"`
		*LocalPoolStorage
		*AzureEndpoint
	}

	if pool.Azure != nil {
		wrapper.Type = "azure"
		wrapper.AzureEndpoint = pool.Azure
	} else if pool.Local.Path != "" {
		wrapper.Type = "local"
		wrapper.LocalPoolStorage = pool.Local
	}

	return json.Marshal(wrapper)
}

func (pool PackagePoolStorage) MarshalYAML() (interface{}, error) {
	var wrapper struct {
		Type              string `yaml:"type,omitempty"`
		*LocalPoolStorage `yaml:",inline"`
		*AzureEndpoint    `yaml:",inline"`
	}

	if pool.Azure != nil {
		wrapper.Type = "azure"
		wrapper.AzureEndpoint = pool.Azure
	} else if pool.Local.Path != "" {
		wrapper.Type = "local"
		wrapper.LocalPoolStorage = pool.Local
	}

	return wrapper, nil
}

// FileSystemPublishRoot describes single filesystem publishing entry point
type FileSystemPublishRoot struct {
	RootDir      string `json:"rootDir"       yaml:"root_dir"`
	LinkMethod   string `json:"linkMethod"    yaml:"link_method"`
	VerifyMethod string `json:"verifyMethod"  yaml:"verify_method"`
}

// S3PublishRoot describes single S3 publishing entry point
type S3PublishRoot struct {
	Region                  string `json:"region"                     yaml:"region"`
	Bucket                  string `json:"bucket"                     yaml:"bucket"`
	Prefix                  string `json:"prefix"                     yaml:"prefix"`
	ACL                     string `json:"acl"                        yaml:"acl"`
	AccessKeyID             string `json:"awsAccessKeyID"             yaml:"access_key_id"`
	SecretAccessKey         string `json:"awsSecretAccessKey"         yaml:"secret_access_key"`
	SessionToken            string `json:"awsSessionToken"            yaml:"session_token"`
	Endpoint                string `json:"endpoint"                   yaml:"endpoint"`
	StorageClass            string `json:"storageClass"               yaml:"storage_class"`
	EncryptionMethod        string `json:"encryptionMethod"           yaml:"encryption_method"`
	PlusWorkaround          bool   `json:"plusWorkaround"             yaml:"plus_workaround"`
	DisableMultiDel         bool   `json:"disableMultiDel"            yaml:"disable_multidel"`
	ForceSigV2              bool   `json:"forceSigV2"                 yaml:"force_sigv2"`
	ForceVirtualHostedStyle bool   `json:"forceVirtualHostedStyle"    yaml:"force_virtualhosted_style"`
	Debug                   bool   `json:"debug"                      yaml:"debug"`
}

// SwiftPublishRoot describes single OpenStack Swift publishing entry point
type SwiftPublishRoot struct {
	Container      string `json:"container"       yaml:"container"`
	Prefix         string `json:"prefix"          yaml:"prefix"`
	UserName       string `json:"osname"          yaml:"username"`
	Password       string `json:"password"        yaml:"password"`
	Tenant         string `json:"tenant"          yaml:"tenant"`
	TenantID       string `json:"tenantid"        yaml:"tenant_id"`
	Domain         string `json:"domain"          yaml:"domain"`
	DomainID       string `json:"domainid"        yaml:"domain_id"`
	TenantDomain   string `json:"tenantdomain"    yaml:"tenant_domain"`
	TenantDomainID string `json:"tenantdomainid"  yaml:"tenant_domain_id"`
	AuthURL        string `json:"authurl"         yaml:"auth_url"`
}

// AzureEndpoint describes single Azure publishing entry point
type AzureEndpoint struct {
	Container   string `json:"container"    yaml:"container"`
	Prefix      string `json:"prefix"       yaml:"prefix"`
	AccountName string `json:"accountName"  yaml:"account_name"`
	AccountKey  string `json:"accountKey"   yaml:"account_key"`
	Endpoint    string `json:"endpoint"     yaml:"endpoint"`
}

// Config is configuration for aptly, shared by all modules
var Config = ConfigStructure{
	RootDir:                filepath.Join(os.Getenv("HOME"), ".aptly"),
	DownloadConcurrency:    4,
	DownloadLimit:          0,
	Downloader:             "default",
	DatabaseOpenAttempts:   -1,
	Architectures:          []string{},
	DepFollowSuggests:      false,
	DepFollowRecommends:    false,
	DepFollowAllVariants:   false,
	DepFollowSource:        false,
	GpgProvider:            "gpg",
	GpgDisableSign:         false,
	GpgDisableVerify:       false,
	GpgKeys:                []string{},
	DownloadSourcePackages: false,
	PackagePoolStorage: PackagePoolStorage{
		Local: &LocalPoolStorage{Path: ""},
	},
	SkipLegacyPool:         false,
	PpaDistributorID:       "ubuntu",
	PpaCodename:            "",
	FileSystemPublishRoots: map[string]FileSystemPublishRoot{},
	S3PublishRoots:         map[string]S3PublishRoot{},
	SwiftPublishRoots:      map[string]SwiftPublishRoot{},
	AzurePublishRoots:      map[string]AzureEndpoint{},
	AsyncAPI:               false,
	EnableMetricsEndpoint:  false,
	LogLevel:               "info",
	LogFormat:              "default",
	ServeInAPIMode:         false,
	EnableSwaggerEndpoint:  false,
}

// LoadConfig loads configuration from json file
func LoadConfig(filename string, config *ConfigStructure) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	decJSON := json.NewDecoder(JsonConfigReader.New(f))
	if err = decJSON.Decode(&config); err != nil {
		_, _ = f.Seek(0, 0)
		decYAML := yaml.NewDecoder(f)
		if err2 := decYAML.Decode(&config); err2 != nil {
			err = fmt.Errorf("invalid yaml (%s) or json (%s)", err2, err)
		} else {
			err = nil
		}
	}
	return err
}

// SaveConfig write configuration to json file
func SaveConfig(filename string, config *ConfigStructure) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	encoded, err := json.MarshalIndent(&config, "", "  ")
	if err != nil {
		return err
	}

	_, err = f.Write(encoded)
	return err
}

// SaveConfigRaw write configuration to file
func SaveConfigRaw(filename string, conf []byte) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	_, err = f.Write(conf)
	return err
}

// SaveConfigYAML write configuration to yaml file
func SaveConfigYAML(filename string, config *ConfigStructure) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	yamlData, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("error marshaling to YAML: %s", err)
	}

	_, err = f.Write(yamlData)
	return err
}

// GetRootDir returns the RootDir with expanded ~ as home directory
func (conf *ConfigStructure) GetRootDir() string {
	return strings.Replace(conf.RootDir, "~", os.Getenv("HOME"), 1)
}
