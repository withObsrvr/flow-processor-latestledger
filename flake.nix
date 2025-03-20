{
  description = "Obsrvr Flow Plugin: Latest Ledger Processor";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages = {
          default = pkgs.buildGoModule {
            pname = "flow-latest-ledger";
            version = "1.0.0";
            src = ./.;
            
            # Start with empty string to calculate hash on first build
            vendorHash = null;
            
            # Enable CGO which is required for Go plugins
            hardeningDisable = [ "all" ];
            
            env = {
              CGO_ENABLED = "1"; # This is critical for plugin mode
              GOPROXY = "direct";
            };
            
            # Build as a shared library/plugin
            buildPhase = ''
              runHook preBuild
              go build -buildmode=plugin -o flow-latest-ledger.so .
              runHook postBuild
            '';

            # Custom install phase for the plugin
            installPhase = ''
              runHook preInstall
              mkdir -p $out/lib
              cp flow-latest-ledger.so $out/lib/
              mkdir -p $out/share
              cp go.mod $out/share/
              if [ -f go.sum ]; then
                cp go.sum $out/share/
              fi
              runHook postInstall
            '';
            
            # Add any C dependencies if needed
            nativeBuildInputs = [ pkgs.pkg-config ];
            buildInputs = [
              # If you need any C libraries, add them here
            ];
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [ 
            # Using Go 1.21 which is widely supported for plugins
            go_1_21
            pkg-config
            gopls
            delve
            # Any additional development tools
          ];
          
          # Enable CGO in the development shell
          env = {
            CGO_ENABLED = "1";
          };
        };
      }
    );
} 