{
  pkgs ? import <nixpkgs> { },
  lib ? pkgs.lib,
  ...
}:
let
  api-schema = pkgs.stdenv.mkDerivation {
    name = "git-bug-api-schema";
    src = ../api/graphql/schema;

    installPhase = ''
      cp -r $src $out
    '';
  };
in
{
  default = pkgs.buildNpmPackage {
    name = "git-bug-web-ui";
    npmDepsHash = "sha256-Mut8+2kiJgeM9KnKctFttQxeq7y0VkUCgpP0zo18Gxg=";

    dontStrip = false;

    src = pkgs.nix-gitignore.gitignoreRecursiveSource [ ./.gitignore ./src/.gitignore ] (
      lib.fileset.toSource {
        root = ./.;
        fileset = lib.fileset.difference ./. (
          lib.fileset.unions [
            (lib.fileset.fileFilter (
              f:
              lib.any (ext: f.hasExt ext) [
                # keep-sorted start
                "go"
                "md"
                "nix"
                # keep-sorted end
              ]
            ) ./.)

            # keep-sorted start
            ./.gitignore
            ./Makefile
            ./src/.gitignore
            # keep-sorted end
          ]
        );
      }
    );

    nativeBuildInputs = with pkgs; [ go ];

    postUnpack = ''
      mkdir -p api/graphql
      cp -r "${api-schema}" "api/graphql/schema"
    '';

    postInstall = ''
      pwd
      cp -rv build/ $out
    '';

    meta = {
      description = "Web interface for git-bug, a distributed issue tracker";
      homepage = "https://github.com/git-bug/git-bug";
      license = lib.licenses.gpl3Plus;
    };
  };
}
