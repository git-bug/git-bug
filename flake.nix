{
  description = "workspace configuration for git-bug";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      flake-utils,
      nixpkgs,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShell = pkgs.mkShell {
          packages = with pkgs; [
            codespell
            delve
            gh
            git
            go
            golangci-lint
            nixfmt-rfc-style
            nodePackages.prettier
            nodejs
            pnpm
          ];

          shellHook = ''
            # Use //:.gitmessage as the commit message template
            ${pkgs.git}/bin/git config --local commit.template ".gitmessage"

            # Use a common, shared file as the default for running
            # git-blame with the `--ignore-revs` flag
            ${pkgs.git}/bin/git config --local blame.ignoreRevsFile ".git-blame-ignore-revs"
          '';
        };
      }
    );
}
