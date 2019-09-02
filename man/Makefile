RUBYBINPATH=$(shell ruby -r rubygems -e 'puts Gem.user_dir')/bin

all: $(RUBYBINPATH)/ronn generate

$(RUBYBINPATH)/ronn:
	gem install --user-install specific_install
	gem specific_install --user-install -l smira/ronn

generate:
	PATH=$(RUBYBINPATH):$(PATH) go run ../_man/gen.go

.PHONY: generate
