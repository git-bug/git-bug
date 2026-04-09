let
  pkgs = import (import ./.npins).nixpkgs { };
  ci.shell = (import ./ci { inherit pkgs; }).shell;
in
pkgs.mkShellNoCC {
  # because we're using mkShellNoCC, we should disable CGO in order to avoid
  # errors when running various go tools (e.g. go generate and go build), which
  # use cgo by default.
  #
  # we're able to do this because git-bug does not have any direct or indirect
  # dependencies on cgo.
  CGO_ENABLED = 0;

  # this ensures that nix commands use the pinned nixpkgs.
  NIX_PATH = builtins.concatStringsSep ":" [ "nixpkgs=${builtins.storePath pkgs.path}" ];

  NPINS_DIRECTORY = toString ./.npins;

  inputsFrom = [ ci.shell ];

  nativeBuildInputs = with pkgs; [
    nixVersions.latest
    npins
    delve
    gh
    gitMinimal
    git-cliff
    go
    golangci-lint
    gopls
    nodejs
    pinact
  ];

  shellHook = builtins.readFile ./shell-hook.bash;
}
