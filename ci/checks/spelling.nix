{
  pkgs,
  src ? ./../..,
}:

pkgs.runCommand "check-spelling"
  {
    inherit src;
    nativeBuildInputs = with pkgs; [ codespell ];
    description = "Check for spelling mistakes";
  }
  ''
    cd "$src"
    codespell --disable-colors */** | tee $out
  ''
