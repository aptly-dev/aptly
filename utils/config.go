package utils

import (
	"encoding/json"
	"os"
)

// ConfigStructure is structure of main configuration
type ConfigStructure struct {
	RootDir             string `json:"rootDir"`
	DownloadConcurrency int    `json:"downloadConcurrency"`
}

// Config is configuration for aptly, shared by all modules
var Config = ConfigStructure{
	RootDir:             "/var/aptly",
	DownloadConcurrency: 4,
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
