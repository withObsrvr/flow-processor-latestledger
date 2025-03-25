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
            
            # Since we're using the vendor directory directly
            vendorHash = null;
            
            # Enable CGO which is required for Go plugins
            hardeningDisable = [ "all" ];
            
            # Configure build environment
            preBuild = ''
              export CGO_ENABLED=1
            '';
            
            # Build as a shared library/plugin
            buildPhase = ''
              runHook preBuild
              # Use -mod=vendor to use the vendor directory
              go build -mod=vendor -buildmode=plugin -o flow-latest-ledger.so .
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
            # Updated to Go 1.24.1
            go_1_24
            pkg-config
            gopls
            delve
            git
            # Any additional development tools
          ];
          
          # Enable CGO in the development shell
          shellHook = ''
            export CGO_ENABLED=1
            
            # Helper to vendor dependencies
            if [ ! -d vendor ]; then
              echo "Vendoring dependencies..."
              go mod tidy
              go mod vendor
            fi
            
            echo "Development environment ready!"
            echo "To build the plugin manually: go build -buildmode=plugin -o flow-latest-ledger.so ."
          '';
        };
      }
    );
} 