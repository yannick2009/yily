{
  description = "A tiny self-hosted secret manager";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
        };

        name = "yily";
        version = "1.0.0";

        devPkgs = with pkgs; [
          air
          go
          golangci-lint
          hivemind
          nilaway
          just
          nodejs
        ];
      in
      {
        devShells = {
          default = pkgs.mkShell {
            buildInputs = devPkgs;
            shellHook = ''
              unset GOROOT
              (cd client && npm install --silent)
              (cd server && go mod tidy)
              echo "welcome to the ${name} v${version} dev shell!"
            '';
          };
        };

        packages = {
          default = {};
          docker = {};
        };
      }
    );
}