alias s := start
alias l := lint
alias t := test
alias fn := format-nix

alias bd := build-docker
alias bc := build-cli

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
    cd server && go test ./...

# format the nix code
format-nix:
   nixfmt ./*.nix

# build the docker image
build-docker:
    nix build .#docker
    docker load < result
    rm -rf result

# build the cli binary
build-cli:
    nix build .#cli
