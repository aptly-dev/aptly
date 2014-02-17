package utils

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ConfigStructure is structure of main configuration
type ConfigStructure struct {
	RootDir                string   `json:"rootDir"`
	DownloadConcurrency    int      `json:"downloadConcurrency"`
	Architectures          []string `json:"architectures"`
	DepFollowSuggests      bool     `json:"dependencyFollowSuggests"`
	DepFollowRecommends    bool     `json:"dependencyFollowRecommends"`
	DepFollowAllVariants   bool     `json:"dependencyFollowAllVariants"`
	DepFollowSource        bool     `json:"dependencyFollowSource"`
	GpgDisableSign         bool     `json:"gpgDisableSign"`
	GpgDisableVerify       bool     `json:"gpgDisableVerify"`
	DownloadSourcePackages bool     `json:"downloadSourcePackages"`
}

// Config is configuration for aptly, shared by all modules
var Config = ConfigStructure{
	RootDir:                filepath.Join(os.Getenv("HOME"), ".aptly"),
	DownloadConcurrency:    4,
	Architectures:          []string{},
	DepFollowSuggests:      false,
	DepFollowRecommends:    false,
	DepFollowAllVariants:   false,
	DepFollowSource:        false,
	GpgDisableSign:         false,
	GpgDisableVerify:       false,
	DownloadSourcePackages: false,
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
