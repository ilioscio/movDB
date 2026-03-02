{
  description = "movdb — movie directory list generator (print-ready HTML/PDF output)";

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
          version = "1.0.0";
          src = ./.;

          # No external Go dependencies → null is correct here.
          # If you add dependencies in the future, replace with the output of:
          #   nix build 2>&1 | grep "got:"
          vendorHash = null;
        };

        # movdb-pdf: one-shot wrapper that generates a PDF directly.
        # Usage: movdb-pdf [movdb-flags] <movies-directory>
        # Use -o list.pdf to set the output path (default: list.pdf).
        # All other flags (-title, -date, -y) are forwarded to movdb.
        movdb-pdf = pkgs.writeShellScriptBin "movdb-pdf" ''
          output="list.pdf"
          args=()
          while [[ $# -gt 0 ]]; do
            case "$1" in
              -o) output="$2"; shift 2 ;;
              *)  args+=("$1"); shift ;;
            esac
          done
          export TYPST_FONT_PATHS="${pkgs.libertine}/share/fonts/opentype/public"
          ${movdb}/bin/movdb -fmt typst "''${args[@]}" \
            | ${pkgs.typst}/bin/typst compile - "$output"
        '';

      in {
        # `nix build` → builds the binary
        packages = {
          default = movdb;
          movdb = movdb;
          pdf = movdb-pdf;
        };

        # `nix run` → run directly
        apps = {
          default = flake-utils.lib.mkApp { drv = movdb; };
          movdb   = flake-utils.lib.mkApp { drv = movdb; };
          pdf     = flake-utils.lib.mkApp { drv = movdb-pdf; };
        };

        # `nix develop` → dev shell with Go toolchain
        devShells.default = pkgs.mkShell {
          buildInputs = [ pkgs.go pkgs.gopls pkgs.gotools pkgs.typst pkgs.libertine ];
          # Typst doesn't automatically pick up fonts from nix store paths;
          # TYPST_FONT_PATHS tells it exactly where to look.
          shellHook = ''
            export TYPST_FONT_PATHS="${pkgs.libertine}/share/fonts/opentype/public"
          '';
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
