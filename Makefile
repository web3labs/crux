# The import path is where your repository can be found.
# To import subpackages, always prepend the full import path.
# If you change this, run `make clean`. Read more: https://git.io/vM7zV
IMPORT_PATH := github.com/blk-io/crux

# V := 1 # When V is set, print commands and build progress.

# Space separated patterns of packages to skip in list, test, format.
IGNORED_PACKAGES := /vendor/

.PHONY: all
all: build

protofiles: proto grpc

proto:
	$Q protoc -I server/ \
		-I $(GOPATH)/../vendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--go_out=plugins=grpc:server \
		server/*.proto

grpc:
	$Q protoc -I server/ \
		-I $(GOPATH)/../vendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--grpc-gateway_out=logtostderr=true:server \
		server/grpc.proto


.PHONY: build
build: protofiles .GOPATH/.ok
	$Q go install $(if $V,-v) $(VERSION_FLAGS) $(IMPORT_PATH)

### Code not in the repository root? Another binary? Add to the path like this.
# .PHONY: otherbin
# otherbin: .GOPATH/.ok
# 	$Q go install $(if $V,-v) $(VERSION_FLAGS) $(IMPORT_PATH)/cmd/otherbin

##### =====> Utility targets <===== #####

.PHONY: clean test list cover format

clean:
	$Q rm -rf bin .GOPATH
	$Q rm -f server/*\.pb*

test: .GOPATH/.ok
	$Q go test $(if $V,-v) -i -race $(allpackages) # install -race libs to speed up next run
ifndef CI
	$Q go vet $(allpackages)
	$Q GODEBUG=cgocheck=2 go test -race $(allpackages)
else
	$Q ( go vet $(allpackages); echo $$? ) | \
	    tee .GOPATH/test/vet.txt | sed '$$ d'; exit $$(tail -1 .GOPATH/test/vet.txt)
	$Q ( GODEBUG=cgocheck=2 go test -v -race $(allpackages); echo $$? ) | \
	    tee .GOPATH/test/output.txt | sed '$$ d'; exit $$(tail -1 .GOPATH/test/output.txt)
endif

list: .GOPATH/.ok
	@echo $(allpackages)

cover: bin/gocovmerge .GOPATH/.ok
	@echo "NOTE: make cover does not exit 1 on failure, don't use it to check for tests success!"
	$Q rm -f .GOPATH/cover/*.out .GOPATH/cover/all.merged
	$(if $V,@echo "-- go test -coverpkg=./... -coverprofile=.GOPATH/cover/... ./...")
	@for MOD in $(allpackages); do \
		go test -coverpkg=`echo $(allpackages)|tr " " ","` \
			-coverprofile=.GOPATH/cover/unit-`echo $$MOD|tr "/" "_"`.out \
			$$MOD 2>&1 | grep -v "no packages being tested depend on"; \
	done
	$Q ./bin/gocovmerge .GOPATH/cover/*.out > .GOPATH/cover/all.merged
ifndef CI
	$Q go tool cover -html .GOPATH/cover/all.merged
else
	$Q go tool cover -html .GOPATH/cover/all.merged -o .GOPATH/cover/all.html
endif
	@echo ""
	@echo "=====> Total test coverage: <====="
	@echo ""
	$Q go tool cover -func .GOPATH/cover/all.merged

format: bin/goimports .GOPATH/.ok
	$Q find .GOPATH/src/$(IMPORT_PATH)/ -iname \*.go | grep -v \
	    -e "^$$" $(addprefix -e ,$(IGNORED_PACKAGES)) | xargs ./bin/goimports -w

##### =====> Internals <===== #####

.PHONY: setup
setup: clean .GOPATH/.ok
	@if ! grep "/.GOPATH" .gitignore > /dev/null 2>&1; then \
	    echo "/.GOPATH" >> .gitignore; \
	    echo "/bin" >> .gitignore; \
	fi
	$Q curl -OL https://github.com/google/protobuf/releases/download/v3.5.1/protoc-3.5.1-linux-x86_64.zip
	$Q unzip protoc-3.5.1-linux-x86_64.zip -d protoc3
	$Q mv protoc3/bin/* .GOPATH/bin/
	$Q mv protoc3/include/* .GOPATH/include/
	$Q rm -r protoc3
	$Q rm protoc-3.5.1-linux-x86_64.zip
	go get -u github.com/golang/dep/cmd/dep
	go get -u golang.org/x/tools/cmd/goimports
	go get -u github.com/wadey/gocovmerge
	go get -u github.com/golang/protobuf/protoc-gen-go
	go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
	@test -f Gopkg.toml || \
		(cd $(CURDIR)/.GOPATH/src/$(IMPORT_PATH) && ./bin/dep init)
	(cd $(CURDIR)/.GOPATH/src/$(IMPORT_PATH) && ./bin/dep ensure)

VERSION          := $(shell git describe --tags --always --dirty="-dev")
DATE             := $(shell date -u '+%Y-%m-%d-%H%M UTC')
VERSION_FLAGS    := -ldflags='-X "main.Version=$(VERSION)" -X "main.BuildTime=$(DATE)"'

# cd into the GOPATH to workaround ./... not following symlinks
_allpackages = $(shell ( cd $(CURDIR)/.GOPATH/src/$(IMPORT_PATH) && \
    GOPATH=$(CURDIR)/.GOPATH go list ./... 2>&1 1>&3 | \
    grep -v -e "^$$" $(addprefix -e ,$(IGNORED_PACKAGES)) 1>&2 ) 3>&1 | \
    grep -v -e "^$$" $(addprefix -e ,$(IGNORED_PACKAGES)))

# memoize allpackages, so that it's executed only once and only if used
allpackages = $(if $(__allpackages),,$(eval __allpackages := $$(_allpackages)))$(__allpackages)

export GOPATH := $(CURDIR)/.GOPATH
unexport GOBIN

Q := $(if $V,,@)

.GOPATH/.ok:
	$Q mkdir -p "$(dir .GOPATH/src/$(IMPORT_PATH))"
	$Q ln -s ../../../.. ".GOPATH/src/$(IMPORT_PATH)"
	$Q mkdir -p .GOPATH/test .GOPATH/cover
	$Q mkdir -p bin
	$Q ln -s ../bin .GOPATH/bin
	$Q touch $@

.PHONY: bin/gocovmerge bin/goimports
bin/gocovmerge: .GOPATH/.ok
	@test -f ./bin/gocovmerge || \
	    { echo "Vendored gocovmerge not found, try running 'make setup'..."; exit 1; }
bin/goimports: .GOPATH/.ok
	@test -d ./vendor/golang.org/x/tools/cmd/goimports || \
	    { echo "Vendored goimports not found, try running 'make setup'..."; exit 1; }
	$Q go install $(IMPORT_PATH)/vendor/golang.org/x/tools/cmd/goimports

