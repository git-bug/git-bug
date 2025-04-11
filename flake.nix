{
  description = "workspace configuration for git-bug";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    utils.url = "github:numtide/flake-utils";
    parts.url = "github:hercules-ci/flake-parts";

    treefmt = {
      url = "github:numtide/treefmt-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    { nixpkgs, ... }@inputs:
    let
      systems = inputs.utils.lib.defaultSystems;
    in
    inputs.parts.lib.mkFlake { inherit inputs; } {
      inherit systems;

      imports = [ inputs.treefmt.flakeModule ];

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
              nodePackages.prettier
            ];

            shellHook = ''
              # Use //:.gitmessage as the commit message template
              ${pkgs.git}/bin/git config --local commit.template ".gitmessage"

              # Use a common, shared file as the default for running
              # git-blame with the `--ignore-revs` flag
              ${pkgs.git}/bin/git config --local blame.ignoreRevsFile ".git-blame-ignore-revs"
            '';
          };
        };
    };
}
