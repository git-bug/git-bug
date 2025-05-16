{
  pkgs,
  excludes ? [ ],
  ...
}:
{
  projectRootFile = "flake.nix";

  programs = {
    gofmt = {
      enable = true;
    };

    mdformat = {
      enable = false;

      package = pkgs.mdformat.withPlugins (
        p: with p; [
          # add support for github flavored markdown
          mdformat-gfm
          mdformat-gfm-alerts

          # add support for markdown tables
          mdformat-tables

          # add the following comment before running `nix fmt` to generate a
          # table of contents in markdown files:
          #     <!-- mdformat-toc start -->
          mdformat-toc
        ]
      );

      settings = {
        end-of-line = "lf";
        number = true;
        wrap = 80;
      };
    };

    nixfmt = {
      enable = true;
      strict = true;
    };

    prettier = {
      enable = true;

      settings = {
        singleQuote = true;
        trailingComma = "es5";
      };
    };

    shfmt = {
      enable = true;
    };

    yamlfmt = {
      enable = true;

      settings.formatter = {
        eof_newline = true;
        include_document_start = true;
        retain_line_breaks_single = true;
        trim_trailing_whitespace = true;
      };
    };
  };

  settings.global.excludes =
    pkgs.lib.lists.unique [
      "*.graphql"
      "*.png"
      "*.svg"
      "*.txt"
      "doc/man/*.1" # generated via //doc:generate.go
      "doc/md/*" # generated via //doc:generate.go
      "misc/completion/*/*"
      "Makefile"
    ]
    ++ excludes;

  settings.formatter = {
    prettier = {
      excludes = [
        "*.md"
        "*.yaml"
        "*.yml"
      ];
    };
  };
}
