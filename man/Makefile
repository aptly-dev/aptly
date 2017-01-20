all: prepare generate

prepare:
	gem install specific_install
	gem specific_install -l smira/ronn

generate:
	cd .. && gom build  -o man/gen man/gen.go
	./gen

.PHONY: prepare generate
