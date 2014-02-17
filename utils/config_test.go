package utils

import (
	. "launchpad.net/gocheck"
	"os"
	"path/filepath"
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
}

func (s *ConfigSuite) TestSaveConfig(c *C) {
	configname := filepath.Join(c.MkDir(), "aptly.json")

	s.config.RootDir = "/tmp/aptly"
	s.config.DownloadConcurrency = 5

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
		"  \"architectures\": null,\n"+
		"  \"dependencyFollowSuggests\": false,\n"+
		"  \"dependencyFollowRecommends\": false,\n"+
		"  \"dependencyFollowAllVariants\": false,\n"+
		"  \"dependencyFollowSource\": false,\n"+
		"  \"gpgDisableSign\": false,\n"+
		"  \"gpgDisableVerify\": false,\n"+
		"  \"downloadSourcePackages\": false\n"+
		"}")
}

const configFile = `{"rootDir": "/opt/aptly/", "downloadConcurrency": 33}`
