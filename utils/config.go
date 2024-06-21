package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ConfigStructure is structure of main configuration
type ConfigStructure struct { // nolint: maligned
	RootDir                string                           `json:"rootDir"`
	DownloadConcurrency    int                              `json:"downloadConcurrency"`
	DownloadLimit          int64                            `json:"downloadSpeedLimit"`
	DownloadRetries        int                              `json:"downloadRetries"`
	Downloader             string                           `json:"downloader"`
	DatabaseOpenAttempts   int                              `json:"databaseOpenAttempts"`
	Architectures          []string                         `json:"architectures"`
	DepFollowSuggests      bool                             `json:"dependencyFollowSuggests"`
	DepFollowRecommends    bool                             `json:"dependencyFollowRecommends"`
	DepFollowAllVariants   bool                             `json:"dependencyFollowAllVariants"`
	DepFollowSource        bool                             `json:"dependencyFollowSource"`
	DepVerboseResolve      bool                             `json:"dependencyVerboseResolve"`
	GpgDisableSign         bool                             `json:"gpgDisableSign"`
	GpgDisableVerify       bool                             `json:"gpgDisableVerify"`
	GpgProvider            string                           `json:"gpgProvider"`
	DownloadSourcePackages bool                             `json:"downloadSourcePackages"`
	PackagePoolStorage     PackagePoolStorage               `json:"packagePoolStorage"`
	SkipLegacyPool         bool                             `json:"skipLegacyPool"`
	PpaDistributorID       string                           `json:"ppaDistributorID"`
	PpaCodename            string                           `json:"ppaCodename"`
	SkipContentsPublishing bool                             `json:"skipContentsPublishing"`
	SkipBz2Publishing      bool                             `json:"skipBz2Publishing"`
	FileSystemPublishRoots map[string]FileSystemPublishRoot `json:"FileSystemPublishEndpoints"`
	S3PublishRoots         map[string]S3PublishRoot         `json:"S3PublishEndpoints"`
	SwiftPublishRoots      map[string]SwiftPublishRoot      `json:"SwiftPublishEndpoints"`
	AzurePublishRoots      map[string]AzureEndpoint         `json:"AzurePublishEndpoints"`
	AsyncAPI               bool                             `json:"AsyncAPI"`
	EnableMetricsEndpoint  bool                             `json:"enableMetricsEndpoint"`
	LogLevel               string                           `json:"logLevel"`
	LogFormat              string                           `json:"logFormat"`
	ServeInAPIMode         bool                             `json:"serveInAPIMode"`
	DatabaseBackend        DBConfig                         `json:"databaseBackend"`
}

// DBConfig
type DBConfig struct {
	Type   string `json:"type"`
	URL    string `json:"url"`
	DbPath string `json:"dbPath"`
}

type LocalPoolStorage struct {
	Path string `json:"path,omitempty"`
}

type PackagePoolStorage struct {
	Local *LocalPoolStorage
	Azure *AzureEndpoint
}

func (pool *PackagePoolStorage) UnmarshalJSON(data []byte) error {
	var discriminator struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(data, &discriminator); err != nil {
		return err
	}

	switch discriminator.Type {
	case "azure":
		pool.Azure = &AzureEndpoint{}
		return json.Unmarshal(data, &pool.Azure)
	case "local", "":
		pool.Local = &LocalPoolStorage{}
		return json.Unmarshal(data, &pool.Local)
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

// FileSystemPublishRoot describes single filesystem publishing entry point
type FileSystemPublishRoot struct {
	RootDir      string `json:"rootDir"`
	LinkMethod   string `json:"linkMethod"`
	VerifyMethod string `json:"verifyMethod"`
}

// S3PublishRoot describes single S3 publishing entry point
type S3PublishRoot struct {
	Region                  string `json:"region"`
	Bucket                  string `json:"bucket"`
	Endpoint                string `json:"endpoint"`
	AccessKeyID             string `json:"awsAccessKeyID"`
	SecretAccessKey         string `json:"awsSecretAccessKey"`
	SessionToken            string `json:"awsSessionToken"`
	Prefix                  string `json:"prefix"`
	ACL                     string `json:"acl"`
	StorageClass            string `json:"storageClass"`
	EncryptionMethod        string `json:"encryptionMethod"`
	PlusWorkaround          bool   `json:"plusWorkaround"`
	DisableMultiDel         bool   `json:"disableMultiDel"`
	ForceSigV2              bool   `json:"forceSigV2"`
	ForceVirtualHostedStyle bool   `json:"forceVirtualHostedStyle"`
	Debug                   bool   `json:"debug"`
}

// SwiftPublishRoot describes single OpenStack Swift publishing entry point
type SwiftPublishRoot struct {
	UserName       string `json:"osname"`
	Password       string `json:"password"`
	AuthURL        string `json:"authurl"`
	Tenant         string `json:"tenant"`
	TenantID       string `json:"tenantid"`
	Domain         string `json:"domain"`
	DomainID       string `json:"domainid"`
	TenantDomain   string `json:"tenantdomain"`
	TenantDomainID string `json:"tenantdomainid"`
	Prefix         string `json:"prefix"`
	Container      string `json:"container"`
}

// AzureEndpoint describes single Azure publishing entry point
type AzureEndpoint struct {
	AccountName string `json:"accountName"`
	AccountKey  string `json:"accountKey"`
	Container   string `json:"container"`
	Prefix      string `json:"prefix"`
	Endpoint    string `json:"endpoint"`
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
	LogLevel:               "debug",
	LogFormat:              "default",
	ServeInAPIMode:         false,
}

// LoadConfig loads configuration from json file
func LoadConfig(filename string, config *ConfigStructure) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	return dec.Decode(&config)
}

// SaveConfig write configuration to json file
func SaveConfig(filename string, config *ConfigStructure) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	encoded, err := json.MarshalIndent(&config, "", "  ")
	if err != nil {
		return err
	}

	_, err = f.Write(encoded)
	return err
}
