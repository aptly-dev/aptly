from lib import BaseTest


class PublishSourceAdd1Test(BaseTest):
    """
    publish source add: add single source
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly snapshot create snap2 empty",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -component=main snap1",
    ]
    runCmd = "aptly publish source add -component=test maverick snap2"
    gold_processor = BaseTest.expand_environ


class PublishSourceAdd2Test(BaseTest):
    """
    publish source add: add multiple sources
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly snapshot create snap2 empty",
        "aptly snapshot create snap3 empty",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -component=main snap1",
    ]
    runCmd = "aptly publish source add -component=test,other-test maverick snap2 snap3"
    gold_processor = BaseTest.expand_environ


class PublishSourceAdd3Test(BaseTest):
    """
    publish source add: (re-)add already added source
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly snapshot create snap2 empty",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -component=main snap1",
    ]
    runCmd = "aptly publish source add -component=main maverick snap2"
    expectedCode = 1
    gold_processor = BaseTest.expand_environ


class PublishSourceList1Test(BaseTest):
    """
    publish source list: show source changes
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly snapshot create snap2 empty",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -component=main snap1",
        "aptly publish source add -component=test maverick snap2",
    ]
    runCmd = "aptly publish source list maverick"
    gold_processor = BaseTest.expand_environ


class PublishSourceList2Test(BaseTest):
    """
    publish source list: show source changes as JSON
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly snapshot create snap2 empty",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -component=main snap1",
        "aptly publish source add -component=test maverick snap2",
    ]
    runCmd = "aptly publish source list -json maverick"
    gold_processor = BaseTest.expand_environ


class PublishSourceList3Test(BaseTest):
    """
    publish source list: show source changes (empty)
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly snapshot create snap2 empty",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -component=main snap1",
    ]
    runCmd = "aptly publish source list maverick"
    expectedCode = 1
    gold_processor = BaseTest.expand_environ


class PublishSourceDrop1Test(BaseTest):
    """
    publish source drop: drop source changes
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly snapshot create snap2 empty",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -component=main snap1",
    ]
    runCmd = "aptly publish source drop maverick"
    gold_processor = BaseTest.expand_environ


class PublishSourceUpdate1Test(BaseTest):
    """
    publish source update: Update single source
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly snapshot create snap2 empty",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -component=main snap1",
    ]
    runCmd = "aptly publish source update -component=main maverick snap2"
    gold_processor = BaseTest.expand_environ


class PublishSourceUpdate2Test(BaseTest):
    """
    publish source update: Update multiple sources
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly snapshot create snap2 empty",
        "aptly snapshot create snap3 empty",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -component=main,test snap1 snap2",
    ]
    runCmd = "aptly publish source update -component=main,test maverick snap2 snap3"
    gold_processor = BaseTest.expand_environ


class PublishSourceUpdate3Test(BaseTest):
    """
    publish source update: Update not existing source
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -component=main snap1",
    ]
    runCmd = "aptly publish source update -component=not-existent maverick snap1"
    gold_processor = BaseTest.expand_environ


class PublishSourceReplace1Test(BaseTest):
    """
    publish source replace: Replace existing sources
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly snapshot create snap2 empty",
        "aptly snapshot create snap3 empty",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -component=main,test snap1 snap2",
    ]
    runCmd = "aptly publish source replace -component=main-new,test-new maverick snap2 snap3"
    gold_processor = BaseTest.expand_environ


class PublishSourceRemove1Test(BaseTest):
    """
    publish source remove: Remove single source
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly snapshot create snap2 empty",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -component=main,test snap1 snap2",
    ]
    runCmd = "aptly publish source remove -component=test maverick"
    gold_processor = BaseTest.expand_environ


class PublishSourceRemove2Test(BaseTest):
    """
    publish source remove: Remove multiple sources
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly snapshot create snap2 empty",
        "aptly snapshot create snap3 empty",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -component=main,test,other-test snap1 snap2 snap3",
    ]
    runCmd = "aptly publish source remove -component=test,other-test maverick"
    gold_processor = BaseTest.expand_environ


class PublishSourceRemove3Test(BaseTest):
    """
    publish source remove: Remove not-existing source
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -component=main snap1",
    ]
    runCmd = "aptly publish source remove -component=not-existent maverick"
    expectedCode = 1
    gold_processor = BaseTest.expand_environ
