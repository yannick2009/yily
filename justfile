alias s := start
alias l := lint
alias t := test
alias c := check
alias fn := format-nix

alias bd := build-docker
alias bc := build-cli
alias r := release

default:
    just --list

# start the client and server
start:
    hivemind

# format & lint the server code
lint:
    (cd client && npx vp check --fix) && \
    (cd server && go fmt ./... && golangci-lint run ./... && go tool nilaway ./...)

# format and check the server code
test:
    (cd server && mkdir -p coverage && go test ./... -race -coverprofile=coverage/coverage.out -covermode=atomic && go tool cover -html=coverage/coverage.out -o coverage/coverage.html)

# check formatting and lint without modifying files
check: lint test

# install the repository git hooks (pre-push runs `just check`)
install-hooks:
    git config core.hooksPath .githooks

# format the nix code
format-nix:
   nixfmt ./*.nix

# build the docker image
build-docker:
    nix build .#docker
    docker load -i result
    rm -rf result

# build the cli binary
build-cli:
    nix build .#cli

# build the cli for all release targets
build-cli-all:
    nix build .#cli-release

# build all release archives
build-release:
    nix build .#release-assets

# tag and push a release; the GitHub workflow builds and publishes the assets
release version:
    chmod +x ./scripts/release.sh
    ./scripts/release.sh "{{version}}"
