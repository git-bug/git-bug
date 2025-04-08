{ pkgs, src }:

pkgs.runCommand "spelling"
  {
    inherit src;
    nativeBuildInputs = with pkgs; [ codespell ];
    description = "Check for spelling mistakes";
  }
  ''
    pushd $src
    codespell --check-hidden */**
    popd
    touch $out
  ''
