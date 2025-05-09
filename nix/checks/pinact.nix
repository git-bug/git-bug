{ pkgs, src }:

pkgs.runCommand "pinact"
  {
    inherit src;
    nativeBuildInputs = with pkgs; [ pinact ];
  }
  ''
    cd "$src"
    pinact run --check --verify
    touch "$out"
  ''
