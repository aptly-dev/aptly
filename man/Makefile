all: prepare generate

prepare:
	gem install specific_install
	gem specific_install -l smira/ronn

generate:
	go run ../_man/gen.go

.PHONY: prepare generate
