{
  description = "Yily is a simple, lightweight, self-hosted alternative to complex secret management platforms. Store and manage API keys, passwords, tokens, and other secrets securely.";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
        };

        name = "yily";
        version = "1.0.0-beta.1"; # Used in package names and the image tag.

        # go.mod targets Go 1.27.
        buildGoModule = pkgs.buildGoModule.override { go = pkgs.go_1_27; };

        # Tools available in `nix develop`.
        devPkgs = with pkgs; [
          air
          go_1_27
          golangci-lint
          hivemind
          just
          nodejs
          nixfmt
        ];

        # Keep build output and local env files out of the Nix source.
        clientSrc = pkgs.lib.cleanSourceWith {
          src = ./client;
          filter =
            path: _type:
            let
              base = baseNameOf (toString path);
            in
            !(builtins.elem base [
              "node_modules"
              "dist"
              "dist-ssr"
            ])
            && !(pkgs.lib.hasPrefix ".env" base);
        };

        # Keep runtime state and secrets out of the Nix source.
        serverSrc = pkgs.lib.cleanSourceWith {
          src = ./server;
          filter =
            path: _type:
            let
              base = baseNameOf (toString path);
            in
            !(builtins.elem base [
              "instance"
              "tmp"
              "coverage"
              "trash"
              "trash.txt"
              "http"
            ])
            && !(pkgs.lib.hasPrefix ".env" base);
        };

        # Build the web client and keep only its dist output.
        clientBundle = pkgs.buildNpmPackage {
          pname = "${name}-client";
          version = version;
          src = clientSrc;
          # Update after changing client/package-lock.json.
          npmDepsHash = "sha256-LW3lAwBIi4kKZUEO7BzhMlMFSmNbLbKwKb+VxSDaCfs=";
          npmInstallFlags = [ "--silent" ];
          installPhase = ''
            mkdir -p $out
            cp -r dist $out/dist
          '';
        };

        # Build the server. SQLite is pure Go, so CGO stays off.
        serverBundle = buildGoModule {
          pname = "${name}-server";
          version = version;
          src = serverSrc;
          # Update after changing server/go.mod or go.sum.
          vendorHash = "sha256-pUMKL38Ck5NzdcsewnZOCEfmwH/RaMP5ckaB2+UWepU=";
          env.CGO_ENABLED = 0;
          subPackages = [ "cmd/server" ];
          ldflags = [
            "-s"
            "-w"
          ];
          postInstall = ''
            mv $out/bin/server $out/bin/${name}
          '';
        };

        # Build the CLI from the same Go module and dependencies.
        cliBundle = buildGoModule {
          pname = "${name}-cli";
          version = version;
          src = serverSrc;
          vendorHash = "sha256-pUMKL38Ck5NzdcsewnZOCEfmwH/RaMP5ckaB2+UWepU=";
          env.CGO_ENABLED = 0;
          subPackages = [ "cmd/cli" ];
          ldflags = [
            "-s"
            "-w"
          ];
          postInstall = ''
            mv $out/bin/cli $out/bin/${name}-cli
          '';
          meta.mainProgram = "${name}-cli"; # Makes `nix run .#cli` work.
        };

        # Bundle the server and put dist next to it.
        yily =
          pkgs.runCommand "${name}-${version}"
            {
              meta = {
                description = "${name} server with the bundled web client";
                mainProgram = name;
              };
            }
            ''
              mkdir -p $out/bin
              cp ${serverBundle}/bin/${name} $out/bin/${name}
              cp -r ${clientBundle}/dist $out/bin/dist
            '';

        # Minimal runtime image. The server is a static Go binary.
        dockerImage = pkgs.dockerTools.buildLayeredImage {
          name = name;
          tag = version;
          contents = [
            yily
            pkgs.cacert # CA bundle exposed through SSL_CERT_FILE.
          ];
          config = {
            Cmd = [ "${yily}/bin/${name}" ];
            WorkingDir = "/data"; # SQLite lives in /data/instance.
            ExposedPorts = {
              "16529/tcp" = { };
            };
            Volumes = {
              "/data" = { }; # Persist the database across containers.
            };
            Env = [
              "SSL_CERT_FILE=${pkgs.cacert}/etc/ssl/certs/ca-bundle.crt" # Expose the CA bundle to the server binary.
            ];
          };
          extraCommands = ''
            mkdir -p data
          '';
        };
      in
      {
        devShells = {
          # `nix develop` installs both toolchains and prepares the dependencies.
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

        # nix build .#default, .#cli or .#docker
        packages = {
          default = yily;
          cli = cliBundle;
          docker = dockerImage;
        };
      }
    );
}
