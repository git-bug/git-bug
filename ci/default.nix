{
  pins ? import ../.npins,
  pkgs ? import (import ../.npins).nixpkgs { },
  lib ? pkgs.lib,
  excludes ? [ ],
  src ? (
    pkgs.nix-gitignore.gitignoreRecursiveSource [
      ../.gitignore
      ../webui/.gitignore
      ../webui/src/.gitignore
    ] ../.
  ),
}:
let
  checksDir = ./checks;

  fmt =
    let
      treefmtEval = (import pins.treefmt).evalModule pkgs {
        projectRootFile = ".git-blame-ignore-revs";

        # be a little more verbsoe to see progress
        settings.verbose = 1;

        # be a little more quiet if unmatched files are encountered (by default
        # this is set to info, which always emits every single unmatched file)
        settings.on-unmatched = "debug";

        # files and directories that should be excluded
        settings.global.excludes =
          lib.lists.unique [
            "*.graphql"
            "*.png"
            "*.svg"
            "*.txt"
            ".npins/*"
            "Makefile"
            "misc/completion/*/*"
            "webui/node_modules/*"
            "webui/build"

            # generated files
            "CHANGELOG.md"
            "doc/man/*.1" # via //doc:generate.go
            "doc/md/*" # via //doc:generate.go
          ]
          ++ excludes;

        # FORMATTERS
        #######################################################################

        programs.gofmt.enable = true;
        programs.shfmt.enable = true;
        programs.zizmor.enable = true;
        programs.nixf-diagnose.enable = true;
        programs.statix.enable = true;

        programs.mdformat = {
          enable = true;

          plugins =
            ps: with ps; [
              # add support for github flavored markdown
              mdformat-gfm
              mdformat-gfm-alerts

              # add the following comment before running the formatter to
              # generate a table of contents in markdown files:
              #     <!-- mdformat-toc start -->
              mdformat-toc
            ];

          settings = {
            end-of-line = "lf";
            number = true;
            wrap = 80;
          };
        };

        programs.nixfmt = {
          enable = true;
          strict = true;
        };

        # programs.biome = {
        #   enable = true;
        #   settings.formatter = {
        #     useEditorconfig = true;
        #   };
        #   settings.javascript.formatter = {
        #     quoteStyle = "single";
        #     semicolons = "always";
        #   };
        #   settings.json.formatter.enabled = false;
        # };
        # settings.formatter.biome.excludes = [ "*.min.js" ];

        programs.keep-sorted.enable = true;

        programs.yamlfmt = {
          enable = true;

          settings.formatter = {
            eof_newline = true;
            include_document_start = true;
            retain_line_breaks_single = true;
            trim_trailing_whitespace = true;
          };
        };

        settings.formatter.markdown-code-runner = {
          command = lib.getExe pkgs.markdown-code-runner;
          options =
            let
              config = pkgs.writers.writeTOML "markdown-code-runner-config" {
                presets.nixfmt = {
                  language = "nix";
                  command = [ (lib.getExe pkgs.nixfmt) ];
                };

                presets.go = {
                  language = "go";
                  command = [
                    "${pkgs.go}/bin/gofmt"
                    "{file}"
                  ];
                  input_mode = "file";
                };

                presets.bash = {
                  language = "bash";
                  command = [
                    (lib.getExe pkgs.shfmt)
                    "--binary-next-line"
                    "--case-indent"
                    "--simplify"
                    "--space-redirects"
                    "-"
                  ];
                };
              };
            in
            [ "--config=${config}" ];
          includes = [ "*.md" ];
          excludes = [ "doc/md/*" ];
        };
      };

      fs = lib.fileset;
      checkFiles = fs.toSource {
        root = ../.;
        fileset = fs.difference ../. (fs.maybeMissing ../.git);
      };
    in
    {
      shell = treefmtEval.config.build.devShell;
      check = treefmtEval.config.build.check checkFiles;
    };
in
{
  # the treefmt shell is exported because the treefmt package is added to the
  # development shell via //:shell.nix
  inherit (fmt) shell;

  # checks is an attrset built from aggregating all of the non-golang tests
  # (defined in //ci/checks). these are tests for things like formatting and
  # spelling.
  checks = {
    fmt = fmt.check;
  }
  // lib.listToAttrs (
    map
      (file: {
        name = lib.removeSuffix ".nix" file;
        value = import (checksDir + "/${file}") { inherit pkgs src; };
      })
      (
        builtins.attrNames (
          lib.filterAttrs (name: type: type == "regular" && lib.hasSuffix ".nix" name) (
            builtins.readDir checksDir
          )
        )
      )
  );
}
