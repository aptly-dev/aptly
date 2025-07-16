package aptly

import (
	. "gopkg.in/check.v1"
)

type InterfacesSuite struct{}

var _ = Suite(&InterfacesSuite{})

func (s *InterfacesSuite) TestBarTypeValues(c *C) {
	// Test that BarType enum values are as expected
	c.Check(int(BarGeneralBuildPackageList), Equals, 0)
	c.Check(int(BarGeneralVerifyDependencies), Equals, 1)
	c.Check(int(BarGeneralBuildFileList), Equals, 2)
	c.Check(int(BarCleanupBuildList), Equals, 3)
	c.Check(int(BarCleanupDeleteUnreferencedFiles), Equals, 4)
	c.Check(int(BarMirrorUpdateDownloadIndexes), Equals, 5)
	c.Check(int(BarMirrorUpdateDownloadPackages), Equals, 6)
	c.Check(int(BarMirrorUpdateBuildPackageList), Equals, 7)
	c.Check(int(BarMirrorUpdateImportFiles), Equals, 8)
	c.Check(int(BarMirrorUpdateFinalizeDownload), Equals, 9)
	c.Check(int(BarPublishGeneratePackageFiles), Equals, 10)
	c.Check(int(BarPublishFinalizeIndexes), Equals, 11)
}
