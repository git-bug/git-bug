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
      "webui/*" # not currently supported, //webui is not yet flakeified
      "Makefile"
    ]
    ++ excludes;
}
