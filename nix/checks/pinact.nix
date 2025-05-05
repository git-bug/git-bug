{ pkgs, src }:

pkgs.writeShellApplication {
  name = "pinact";
  runtimeInputs = with pkgs; [ pinact ];
  text = "pinact run --check --verify";
}
