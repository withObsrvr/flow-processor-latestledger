{
  description = "Obsrvr Flow Plugin: Latest-Ledger Processor";

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
            
            # Initially empty, will be updated after first build attempt
            vendorHash = null;
            
            # Build configuration
            env = {
              # No CGO needed for this plugin
              CGO_ENABLED = "0";
              # Allow Go to download modules directly if needed
              GOPROXY = "direct";
            };
            
            # Build as a plugin, ignoring vendor directory
            buildPhase = ''
              runHook preBuild
              go build -mod=mod -buildmode=plugin -o flow-latest-ledger.so .
              runHook postBuild
            '';

            # Install the plugin
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
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [ 
            go_1_21  # Adjust to match your project's Go version
            gopls
            delve
          ];
        };
      }
    );
} 