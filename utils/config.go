package utils

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ConfigStructure is structure of main configuration
type ConfigStructure struct {
	RootDir                string                   `json:"rootDir"`
	DownloadConcurrency    int                      `json:"downloadConcurrency"`
	DownloadLimit          int64                    `json:"downloadSpeedLimit"`
	Architectures          []string                 `json:"architectures"`
	DepFollowSuggests      bool                     `json:"dependencyFollowSuggests"`
	DepFollowRecommends    bool                     `json:"dependencyFollowRecommends"`
	DepFollowAllVariants   bool                     `json:"dependencyFollowAllVariants"`
	DepFollowSource        bool                     `json:"dependencyFollowSource"`
	GpgDisableSign         bool                     `json:"gpgDisableSign"`
	GpgDisableVerify       bool                     `json:"gpgDisableVerify"`
	DownloadSourcePackages bool                     `json:"downloadSourcePackages"`
	PpaDistributorID       string                   `json:"ppaDistributorID"`
	PpaCodename            string                   `json:"ppaCodename"`
	S3PublishRoots         map[string]S3PublishRoot `json:"S3PublishEndpoints"`
}

// S3PublishRoot describes single S3 publishing entry point
type S3PublishRoot struct {
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
	AccessKeyID     string `json:"awsAccessKeyID"`
	SecretAccessKey string `json:"awsSecretAccessKey"`
	Prefix          string `json:"prefix"`
	ACL             string `json:"acl"`
}

// Config is configuration for aptly, shared by all modules
var Config = ConfigStructure{
	RootDir:                filepath.Join(os.Getenv("HOME"), ".aptly"),
	DownloadConcurrency:    4,
	DownloadLimit:          0,
	Architectures:          []string{},
	DepFollowSuggests:      false,
	DepFollowRecommends:    false,
	DepFollowAllVariants:   false,
	DepFollowSource:        false,
	GpgDisableSign:         false,
	GpgDisableVerify:       false,
	DownloadSourcePackages: false,
	PpaDistributorID:       "ubuntu",
	PpaCodename:            "",
	S3PublishRoots:         map[string]S3PublishRoot{},
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
