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
        # Single source of truth for package names and the image tag.
        version = pkgs.lib.removeSuffix "\n" (builtins.readFile ./VERSION);
        commit = if self ? dirtyShortRev then "${self.dirtyShortRev}-dirty" else self.shortRev or "unknown";
        buildDate = self.lastModifiedDate or "unknown";

        # Build metadata injected into the server and every CLI binary.
        versionLdflags = [
          "-s"
          "-w"
          "-X"
          "github.com/yannick2009/yily/internal/version.Number=${version}"
          "-X"
          "github.com/yannick2009/yily/internal/version.Commit=${commit}"
          "-X"
          "github.com/yannick2009/yily/internal/version.Date=${buildDate}"
        ];

        # Cross-compile the CLI for every target platform. The Go toolchain is used to build the CLI, so CGO stays off.
        cliTargets = [
          {
            label = "linux_amd64";
            goos = "linux";
            goarch = "amd64";
          }
          {
            label = "linux_arm64";
            goos = "linux";
            goarch = "arm64";
          }
          {
            label = "linux_armv7";
            goos = "linux";
            goarch = "arm";
            goarm = 7;
          }
          {
            label = "darwin_amd64";
            goos = "darwin";
            goarch = "amd64";
          }
          {
            label = "darwin_arm64";
            goos = "darwin";
            goarch = "arm64";
          }
          {
            label = "windows_amd64";
            goos = "windows";
            goarch = "amd64";
          }
          {
            label = "windows_arm64";
            goos = "windows";
            goarch = "arm64";
          }
        ];

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

        # Shared Go build inputs for the server and CLI targets.
        goCommon = {
          version = version;
          src = serverSrc;
          # Update after changing server/go.mod or go.sum.
          vendorHash = "sha256-pUMKL38Ck5NzdcsewnZOCEfmwH/RaMP5ckaB2+UWepU=";
          env.CGO_ENABLED = 0;
          ldflags = versionLdflags;
        };

        # SQLite is pure Go, so CGO stays off.
        serverBundle = buildGoModule (
          goCommon
          // {
            pname = "${name}-server";
            subPackages = [ "cmd/server" ];
            postInstall = ''
              mv $out/bin/server $out/bin/${name}
            '';
          }
        );

        # Build a CLI binary for a specific target platform. The Go toolchain is used to cross-compile the CLI, so CGO stays off.
        mkCli =
          {
            label,
            goos,
            goarch,
            goarm ? null,
            doCheck ? false,
            ...
          }:
          let
            exe = pkgs.lib.optionalString (goos == "windows") ".exe";
          in
          (buildGoModule (
            goCommon
            // {
              pname = "${name}-cli-${label}";
              subPackages = [ "cmd/cli" ];
              inherit doCheck;
            }
          )).overrideAttrs
            (old: {
              # buildGoModule resets GOOS/GOARCH from the host Go derivation.
              # Override the final environment to cross-compile with the same toolchain.
              env =
                old.env
                // {
                  GOOS = goos;
                  GOARCH = goarch;
                }
                // pkgs.lib.optionalAttrs (goarm != null) {
                  GOARM = toString goarm;
                };

              # Cross-compiled `go install` writes to bin/$GOOS_$GOARCH.
              postInstall = ''
                if [ -d "$out/bin/${goos}_${goarch}" ]; then
                  mv "$out/bin/${goos}_${goarch}"/cli* "$out/bin/"
                  rmdir "$out/bin/${goos}_${goarch}"
                fi
                mv "$out/bin/cli${exe}" "$out/bin/${name}-cli${exe}"
              '';

              meta.mainProgram = "${name}-cli${exe}";
            });

        # Build a CLI binary for every target platform. The Go toolchain is used to cross-compile the CLI, so CGO stays off.
        cliPackages = map (
          target:
          target
          // {
            bundle = mkCli target;
            exe = pkgs.lib.optionalString (target.goos == "windows") ".exe";
          }
        ) cliTargets;

        # Native CLI package kept as `nix build .#cli`.
        cliBundle = mkCli {
          label = "${pkgs.go_1_27.GOOS}_${pkgs.go_1_27.GOARCH}";
          goos = pkgs.go_1_27.GOOS;
          goarch = pkgs.go_1_27.GOARCH;
          doCheck = true;
        };

        # CLI archives for every release target.
        cliRelease =
          pkgs.runCommand "${name}-cli-${version}-release"
            {
              nativeBuildInputs = [
                pkgs.zip
                pkgs.gzip
              ];
            }
            ''
              mkdir -p $out

              ${pkgs.lib.concatMapStrings (target: ''
                staging=$TMPDIR/${name}-cli_${version}_${target.label}
                mkdir -p "$staging"
                cp ${target.bundle}/bin/${name}-cli${target.exe} "$staging/"
                cp ${./LICENSE} "$staging/LICENSE"
                cp ${./README.md} "$staging/README.md"
                chmod 0755 "$staging/${name}-cli${target.exe}"
                chmod 0644 "$staging/LICENSE" "$staging/README.md"

                if [ "${target.goos}" = "windows" ]; then
                  (cd "$TMPDIR" && zip -qr "$out/${name}-cli_${version}_${target.label}.zip" \
                    "${name}-cli_${version}_${target.label}")
                else
                  tar --owner=0 --group=0 --numeric-owner -czf "$out/${name}-cli_${version}_${target.label}.tar.gz" \
                    -C "$TMPDIR" "${name}-cli_${version}_${target.label}"
                fi
              '') cliPackages}
            '';

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

        # Server archive for the release assets.
        serverArchive =
          pkgs.runCommand "${name}-server-${version}-linux_amd64"
            {
              nativeBuildInputs = [ pkgs.gzip ];
            }
            ''
              mkdir -p $out
              staging=$TMPDIR/${name}_${version}_linux_amd64
              mkdir -p "$staging"
              cp ${yily}/bin/${name} "$staging/"
              cp -r ${yily}/bin/dist "$staging/dist"
              cp ${./LICENSE} "$staging/LICENSE"
              cp ${./README.md} "$staging/README.md"
              chmod 0755 "$staging/${name}"
              chmod 0644 "$staging/LICENSE" "$staging/README.md"
              tar --owner=0 --group=0 --numeric-owner -czf "$out/${name}_${version}_linux_amd64.tar.gz" \
                -C "$TMPDIR" "${name}_${version}_linux_amd64"
            '';

        # All archives uploaded to the GitHub release.
        releaseAssets =
          pkgs.runCommand "${name}-${version}-release"
            {
              nativeBuildInputs = [ pkgs.coreutils ];
            }
            ''
              mkdir -p $out
              cp ${serverArchive}/*.tar.gz $out/
              cp ${cliRelease}/*.tar.gz $out/
              cp ${cliRelease}/*.zip $out/
              (cd $out && sha256sum ./*.tar.gz ./*.zip > checksums.txt)
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

        # nix build .#default, .#cli, .#cli-release, .#release-assets or .#docker
        packages = {
          default = yily;
          cli = cliBundle;
          cli-release = cliRelease;
          release-assets = releaseAssets;
          docker = dockerImage;
        }
        // pkgs.lib.listToAttrs (
          map (target: {
            name = "cli-${target.label}";
            value = target.bundle;
          }) cliPackages
        );
      }
    );
}
