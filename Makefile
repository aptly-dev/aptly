prepare:
	go get -d -v ./...
	go get launchpad.net/gocheck
	go get github.com/axw/gocov/gocov
	go get github.com/golang/lint/golint
	go get github.com/matm/gocov-html
	go get github.com/mattn/goveralls

coverage:
	gocov test ./... | gocov-html > coverage.html
	open coverage.html

check:
	go tool vet -all=true .
	golint .

test:
	go test -v ./...