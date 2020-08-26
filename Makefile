.PHONY: default mac linux windows docker clean
BINARY_NAME = superdentist-backend
CHART_NAME = superdentist-backend

default: linux

all: linux test swagger

clean:
	-rm $(BINARY_NAME) ;
coverage: ## Run tests and generate coverage files per package
	mkdir .coverage 2> /dev/null || true
	rm -rf .coverage/*.out || true
	@go test -race ./... -coverprofile=coverage.out -covermode=atomic


docker: linux
	@echo "building docker image" ;\
		docker build -t "$(BINARY_NAME):localdeploy" .
mac:
	@echo "building $(BINARY_NAME) (mac)" ;\
        go build -o $(BINARY_NAME) 
linux:
	## CGO_ENABLED=0 go build -a -installsuffix cgo is not needed for Go 1.10 or later
	## https://github.com/golang/go/issues/9344#issuecomment-69944514
	@echo "building $(BINARY_NAME) (linux)" ;\
        GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME) 
windows:
	@echo "building $(BINARY_NAME) (windows)" ;\
      		env GOOS=windows GOARCH=amd64 go build -o $(BINARY_NAME) 
test:
	@CC=gcc go test ./...

test-race:
	@CC=gcc go test -race -short ./...

deps-lint:
	@GO111MODULE=off go get golang.org/x/lint
	@GO111MODULE=off go get golang.org/x/lint/golint
	@GO111MODULE=off go get golang.org/x/tools/cmd/goimports
	@GO111MODULE=off go get golang.org/x/tools/go/analysis/passes/nilness/cmd/nilness

deps-verify:
	@go mod tidy
	@go mod verify

lint:
	@go vet ./...
	@go vet -vettool=$(which nilness) ./...
	@go fix ./...
	@golint ./...
	# @! goimports -l . | grep -vF 'No Exceptions'

