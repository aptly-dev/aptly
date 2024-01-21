aptly -architectures=i386,amd64 mirror create wheezy-main http://mirror.yandex.ru/debian/ wheezy main
aptly -architectures=i386,amd64 mirror create wheezy-contrib http://mirror.yandex.ru/debian/ wheezy contrib
aptly -architectures=i386,amd64 mirror create wheezy-non-free http://mirror.yandex.ru/debian/ wheezy non-free
aptly -architectures=i386,amd64 mirror create wheezy-updates http://mirror.yandex.ru/debian/ wheezy-updates
aptly -architectures=i386,amd64 mirror create wheezy-backports http://mirror.yandex.ru/debian/ wheezy-backports

aptly mirror update wheezy-main
aptly mirror update wheezy-contrib
aptly mirror update wheezy-non-free
aptly mirror update wheezy-updates
aptly mirror update wheezy-backports

aptly -architectures=i386,amd64 mirror create -with-sources wheezy-main-src http://mirror.yandex.ru/debian/ wheezy main
aptly -architectures=i386,amd64 mirror create -with-sources wheezy-contrib-src http://mirror.yandex.ru/debian/ wheezy contrib
aptly -architectures=i386,amd64 mirror create -with-sources wheezy-non-free-src http://mirror.yandex.ru/debian/ wheezy non-free
aptly -architectures=i386,amd64 mirror create -with-sources wheezy-updates-src http://mirror.yandex.ru/debian/ wheezy-updates
aptly -architectures=i386,amd64 mirror create -with-sources wheezy-backports-src http://mirror.yandex.ru/debian/ wheezy-backports

aptly mirror update wheezy-main-src
aptly mirror update wheezy-contrib-src
aptly mirror update wheezy-non-free-src
aptly mirror update wheezy-updates-src
aptly mirror update wheezy-backports-src

aptly mirror create gnuplot-maverick http://repo.aptly.info/system-tests/ppa.launchpad.net/gladky-anton/gnuplot/ubuntu/ maverick
aptly mirror update gnuplot-maverick

aptly mirror create -with-sources gnuplot-maverick-src http://repo.aptly.info/system-tests/ppa.launchpad.net/gladky-anton/gnuplot/ubuntu/ maverick
aptly mirror update gnuplot-maverick-src

aptly mirror create sensu http://repos.sensuapp.org/apt sensu
aptly mirror update sensu
