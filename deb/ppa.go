package deb

import (
	"fmt"
	"github.com/smira/aptly/utils"
	"os/exec"
	"regexp"
	"strings"
)

var ppaRegexp = regexp.MustCompile("^ppa:([^/]+)/(.+)$")

// ParsePPA converts ppa URL like ppa:user/ppa-name to full HTTP url
func ParsePPA(ppaURL string, config *utils.ConfigStructure) (url string, distribution string, components []string, err error) {
	matches := ppaRegexp.FindStringSubmatch(ppaURL)
	if matches == nil {
		err = fmt.Errorf("unable to parse ppa URL: %v", ppaURL)
		return
	}

	distributorID := config.PpaDistributorID
	if distributorID == "" {
		distributorID, err = getDistributorID()
		if err != nil {
			err = fmt.Errorf("unable to figure out Distributor ID: %s, please set config option ppaDistributorID", err)
			return
		}
	}

	codename := config.PpaCodename
	if codename == "" {
		codename, err = getCodename()
		if err != nil {
			err = fmt.Errorf("unable to figure out Codename: %s, please set config option ppaCodename", err)
			return
		}
	}

	distribution = codename
	components = []string{"main"}
	url = fmt.Sprintf("http://ppa.launchpad.net/%s/%s/%s", matches[1], matches[2], distributorID)

	return
}

func getCodename() (string, error) {
	out, err := exec.Command("lsb_release", "-sc").Output()
	return strings.TrimSpace(string(out)), err
}

func getDistributorID() (string, error) {
	out, err := exec.Command("lsb_release", "-si").Output()
	return strings.ToLower(strings.TrimSpace(string(out))), err
}
