{
  description = "workspace configuration for git-bug";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    flake-parts.url = "github:hercules-ci/flake-parts";

    treefmt-nix = {
      url = "github:numtide/treefmt-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    { nixpkgs, ... }@inputs:
    let
      systems = inputs.flake-utils.lib.defaultSystems;
    in
    inputs.flake-parts.lib.mkFlake { inherit inputs; } {
      inherit systems;

      imports = [ inputs.treefmt-nix.flakeModule ];

      perSystem =
        { pkgs, system, ... }:
        {
          treefmt = import ./treefmt.nix { inherit pkgs; };

          devShells.default = pkgs.mkShell {
            packages = with pkgs; [
              codespell
              delve
              gh
              git
              go
              golangci-lint
            ];

            shellHook = builtins.readFile ./flake-hook.bash;
          };
        };
    };
}
