{
  description = "Obsrvr Flow Plugin: Latest Ledger Processor (Dual Target)";

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
          # Native Go plugin (default)
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
            
            # Build as a shared library/plugin with the goplugin tag
            buildPhase = ''
              runHook preBuild
              go build -mod=vendor -tags=goplugin -buildmode=plugin -o flow-latest-ledger.so .
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
          
          # WebAssembly module target
          wasm = pkgs.buildGoModule {
            pname = "flow-latest-ledger-wasm";
            version = "1.0.0";
            src = ./.;
            
            # Since we're using the vendor directory directly
            vendorHash = null;
            
            # Build as a WebAssembly module with the wasmmodule tag
            buildPhase = ''
              runHook preBuild
              export GOOS=wasip1
              export GOARCH=wasm
              go build -mod=vendor -tags=wasmmodule -buildmode=c-shared -o flow-latest-ledger.wasm .
              runHook postBuild
            '';

            # Custom install phase for the WebAssembly module
            installPhase = ''
              runHook preInstall
              mkdir -p $out/lib
              cp flow-latest-ledger.wasm $out/lib/
              mkdir -p $out/share
              cp go.mod $out/share/
              if [ -f go.sum ]; then
                cp go.sum $out/share/
              fi
              runHook postInstall
            '';
          };
        };

        devShells = {
          # Development shell for the Go plugin (default)
          default = pkgs.mkShell {
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
              echo "To build the Go plugin: go build -tags=goplugin -buildmode=plugin -o flow-latest-ledger.so ."
              echo "To build the WASM module: GOOS=wasip1 GOARCH=wasm go build -tags=wasmmodule -buildmode=c-shared -o flow-latest-ledger.wasm ."
            '';
          };
          
          # Development shell specifically for WebAssembly development
          wasm = pkgs.mkShell {
            buildInputs = with pkgs; [ 
              # Updated to Go 1.24.1
              go_1_24
              gopls
              delve
              git
              # Tools for working with WASM
              wasmtime
            ];
            
            # Set up the WASM development environment
            shellHook = ''
              # Helper to vendor dependencies
              if [ ! -d vendor ]; then
                echo "Vendoring dependencies..."
                go mod tidy
                go mod vendor
              fi
              
              echo "WebAssembly development environment ready!"
              echo "To build the WASM module: GOOS=wasip1 GOARCH=wasm go build -tags=wasmmodule -buildmode=c-shared -o flow-latest-ledger.wasm ."
            '';
          };
        };
      }
    );
} 