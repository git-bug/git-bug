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
      enable = true;

      package = pkgs.mdformat.withPlugins (
        p: with p; [
          # add support for github flavored markdown
          mdformat-gfm

          # add support for github flavored markdown "alerts"
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
        number = true;
        wrap = 80;
      };
    };

    nixfmt = {
      enable = true;
      strict = true;
    };

    # this is disabled due to `//webui` not yet being integrated with the flake.
    # the entire package directory is ignored below in
    # `settings.global.excludes`.
    prettier = {
      enable = false;

      settings = {
        singleQuote = true;
        trailingComma = "es5";
      };
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
      "doc/man/*.1"
      "misc/completion/*/*"
      "webui/*"
      "/.envrc"
      "/.envrc.local"
      "/.editorconfig"
      ".codespellrc"
      # ".direnv/*"
      # ".git"
      "Makefile"
    ]
    ++ excludes;
}
