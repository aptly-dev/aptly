RUBYBINPATH=$(shell ruby -r rubygems -e 'puts Gem.user_dir')/bin

all: $(RUBYBINPATH)/ronn generate

$(RUBYBINPATH)/ronn:
	gem install --user-install specific_install
	gem specific_install --user-install -l https://github.com/smira/ronn.git -b 49e19e2409e462a96fd0f2a50ceead07f392893d

generate:
	PATH=$(RUBYBINPATH):$(PATH) go run ../_man/gen.go

.PHONY: generate
