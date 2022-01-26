GO_BIN_FILES=cmd/git/git.go 
#for race CGO_ENABLED=1
# GO_ENV=GOOS=linux CGO_ENABLED=1
GO_ENV=GOOS=linux CGO_ENABLED=0
# GO_BUILD=go build -ldflags '-s -w' -race
GO_BUILD=go build -ldflags '-s -w'
GO_FMT=gofmt -s -w
GO_LINT=golint -set_exit_status
GO_VET=go vet
GO_IMPORTS=goimports -w
GO_ERRCHECK=errcheck -asserts -ignore '[FS]?[Pp]rint*'
BINARIES=git
all: check ${BINARIES}
git: ${GO_BIN_FILES}
	 ${GO_ENV} ${GO_BUILD} -o git ${GO_BIN_FILES}
fmt: ${GO_BIN_FILES}
	${GO_FMT} ${GO_BIN_FILES}
lint: ${GO_BIN_FILES}
	${GO_LINT} ${GO_BIN_FILES}
vet: ${GO_BIN_FILES}
	${GO_VET} ${GO_BIN_FILES}
imports: ${GO_BIN_FILES}
	${GO_IMPORTS} ${GO_BIN_FILES}
errcheck: ${GO_BIN_FILES}
	${GO_ERRCHECK} ${GO_BIN_FILES}
check: fmt lint imports vet errcheck
clean:
	rm -rf ${BINARIES}
.PHONY: all
