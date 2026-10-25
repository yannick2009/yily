{ pkgs, system }:
let
  sources = {
    version = "1.4.0";

    platforms = {
      x86_64-linux = {
        target = "x86_64-unknown-linux-gnu";
        sha256 = "992558abc9621384434a22d80e06e117615680f14a3e17a58eb287fd30a07694";
      };
      aarch64-linux = {
        target = "aarch64-unknown-linux-gnu";
        sha256 = "8eee2d3e9a5bf8654f5150048f5a82b86d4f546c34c530f052545f5058db60de";
      };
      x86_64-darwin = {
        target = "x86_64-apple-darwin";
        sha256 = "10cdffc49c20dea7ef5bd69f74e8a52f6553076e551326d4efd1013b2efdd9fa";
      };
      aarch64-darwin = {
        target = "aarch64-apple-darwin";
        sha256 = "d54d258ce67d43c0deffb09329825241db197c2f20e72871121227f03fb40312";
      };
    };
  };
  platform = sources.platforms.${system};
in
pkgs.stdenv.mkDerivation rec {
  pname = "vite-plus";
  version = sources.version;
  src = pkgs.fetchurl {
    url = "https://github.com/voidzero-dev/vite-plus/releases/download/v${version}/vp-${platform.target}.tar.gz";
    sha256 = platform.sha256;
  };

  sourceRoot = ".";

  nativeBuildInputs = [ pkgs.autoPatchelfHook ];
  buildInputs = [ pkgs.stdenv.cc.cc.lib ];

  installPhase = ''
    runHook preInstall
    install -Dm755 "$(find . -type f -name vp | head -n1)" $out/bin/vp
    runHook postInstall
  '';

  meta = with pkgs.lib; {
    description = "The unified toolchain and entry point for web development";
    homepage = "https://github.com/voidzero-dev/vite-plus";
    license = licenses.mit;
    mainProgram = "vp";
  };
}
