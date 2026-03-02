{
  description = "movdb — movie directory list generator (print-ready HTML output)";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        movdb = pkgs.buildGoModule {
          pname = "movdb";
          version = "0.1.0";
          src = ./.;

          # No external Go dependencies → null is correct here.
          # If you add dependencies in the future, replace with the output of:
          #   nix build 2>&1 | grep "got:"
          vendorHash = null;
        };

      in {
        # `nix build` → builds the binary
        packages = {
          default = movdb;
          movdb = movdb;
        };

        # `nix run` → run directly
        apps = {
          default = flake-utils.lib.mkApp { drv = movdb; };
          movdb   = flake-utils.lib.mkApp { drv = movdb; };
        };

        # `nix develop` → dev shell with Go toolchain
        devShells.default = pkgs.mkShell {
          buildInputs = [ pkgs.go pkgs.gopls pkgs.gotools ];
        };
      })

    # ── NixOS module (for use from other flakes) ──────────────────────────────
    // {
      # Overlay: adds pkgs.movdb on any system
      overlays.default = final: prev: {
        movdb = self.packages.${final.system}.movdb;
      };

      # NixOS module: declarative installation
      # Usage in another flake:
      #   inputs.movdb.url = "github:ilioscio/movDB";
      #   # in nixosConfigurations:
      #   modules = [ movdb.nixosModules.default ];
      #   # then in your config:
      #   programs.movdb.enable = true;
      nixosModules.default = { config, lib, pkgs, ... }:
        let
          cfg = config.programs.movdb;
        in {
          options.programs.movdb = {
            enable = lib.mkEnableOption "movdb movie list generator";
          };

          config = lib.mkIf cfg.enable {
            environment.systemPackages = [
              self.packages.${pkgs.system}.movdb
            ];
          };
        };
    };
}
