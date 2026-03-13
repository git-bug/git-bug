{
  pkgs ? import (import ./.npins).nixpkgs { },
  version ? builtins.null,
  debug ? false,
}:
let
  # releaseTargets is a list of platforms (in the nix double format) to platforms
  # in the `go tool dist list` format (that GOOS and GOOARCH expect). this is
  # used to generate a list of release targets, and to include/exclude tests
  # when building cross-platform and native binaries.
  releaseTargets = {
    aarch64-embedded = "linux/arm64";
    aarch64-multiplatform = "linux/arm64";
    aarch64-windows = "windows/arm64";
    riscv64 = "linux/riscv64";
    riscv64-embedded = "linux/riscv64";
    x86_64-linux = "linux/amd64";
    x86_64-windows = "windows/amd64";
    x86_64-darwin = "darwin/amd64";
    aarch64-darwin = "darwin/arm64";
  };

  inherit (pkgs) lib;

  buildTarget =
    {
      platform ? pkgs.stdenv.hostPlatform.system,
      version ? builtins.null,
      release ? false,
    }:
    let
      parts = lib.strings.splitString "/" releaseTargets."${platform}";
      suffix = lib.strings.concatStringsSep "-" parts;
      goos = builtins.elemAt parts 0;
      goarch = builtins.elemAt parts 1;

      isNative = platform == pkgs.stdenv.hostPlatform.system;
    in
    assert !(debug && release) || throw ''
      building release binaries with the debug flag is not supported.

      building release binaries is time consuming and resource intensive, as
      the drv is built for all supported platforms. as such, it is recommended
      to let the pipeline handle this.

      to build a debugger-friendly binary for your platform, run one of the
      following commands instead:

      - mask build -d
      - make build/debug
    '';
    # git-bug does not currently have any direct or transitive C dependencies.
    # as such, we do not need to use pkgsCross, which would introduce
    # additional overhead, but does properly handle setting up different
    # linkers and would be required if cgo is introduced.
    pkgs.buildGoModule {
      name = if isNative then "git-bug" else "git-bug-${suffix}";

      src = lib.sources.cleanSourceWith {
        filter =
          name: type:
          let
            baseName = baseNameOf (toString name);
          in
          !(
            (
              type == "directory"
              && lib.any (n: n == baseName) [
                # keep-sorted start
                ".direnv"
                ".git"
                ".githooks.d"
                ".github"
                ".npins"
                "nix"
                "webui/node_modules"
                "webui/public"
                "webui/src"
                "webui/types"
                # keep-sorted end
              ]
            )
            || (lib.any (n: n == baseName) [
              # keep-sorted start
              ".codespellrc"
              ".editorconfig"
              ".envrc"
              ".envrc.local"
              ".git-blame-ignore-revs"
              ".gitignore"
              ".gitmessage"
              ".mailmap"
              ".pinact.yaml"
              "Makefile"
              "Maskfile.md"
              "cliff.toml"
              "default.nix"
              "git-bug" # the binary built when using the go compiler directly
              "shell-hook.bash"
              "shell.nix"
              # keep-sorted end
            ])
            # swap, backup, and temporary files
            || lib.strings.hasSuffix "~" baseName
            || lib.strings.match "^\\.sw[a-z]$" baseName != null
            || lib.strings.match "^\\..*\\.sw[a-z]$" baseName != null
            # nix-build result symlinks
            || (type == "symlink" && lib.strings.hasPrefix "result" baseName)
            # misc stuff (sockets and such that don't belong in the store)
            || (type == "unknown")
          );
        src = ./.;
      };

      vendorHash = "sha256-F5E7Xu6t3f4rDaA8izqzR6ha8EHSdiQSHdj/LlVBAj0=";

      excludedPackages = [
        # keep-sorted start
        "doc"
        "misc"
        "tests"
        # keep-sorted end
      ];

      ldflags =
        lib.optionals (!debug || release) [
          # keep-sorted start
          "-s" # omit the symbol table
          "-w" # omit DWARF debug info
          # keep-sorted end
        ]
        ++ (
          let
            versionString = lib.concatStringsSep "-" [
              (lib.optionalString (version != builtins.null) version)
              (lib.optionalString debug "debug")
            ];
          in
            [ "-X main.version=${versionString}" ]
        );

      # normally, pack the web ui for builds, but for debug builds, fetch it
      # from the filesystem (this is what this build tag controls)
      tags = lib.optionals (debug && !release) [ "debug" ];

      flags = lib.optionals (debug && !release) [
        # enable debugger-friendly compiler options for non-release builds
        "-gcflags=all='-N -l'"
      ];

      env = {
        CGO_ENABLED = 0; # disable cgo to enable static builds
      };

      preBuild = lib.strings.concatLines (
        lib.optionals (!debug || release) [
          # TODO: embed //webui
        ]
        ++ [
          # keep-sorted start
          "export GOARCH=${goarch}"
          "export GOOS=${goos}"
          # keep-sorted end
        ]
      );

      postInstall =
        let
          extension = lib.optionalString (goos == "windows") ".exe";
        in
        lib.optionalString release ''
          if test -d $out/bin/${goos}_${goarch}; then
            mv \
              $out/bin/${goos}_${goarch}/git-bug* \
              $out/bin/git-bug-${suffix}${extension}
            rmdir $out/bin/${goos}_${goarch}
          fi

          # the host-native binary when built on linux or macos
          if test -x $out/bin/git-bug; then
            mv $out/bin/git-bug $out/bin/git-bug-${suffix}
          fi
        '';

      nativeCheckInputs = with pkgs; [ gitMinimal ];

      doCheck = isNative && !debug;

      # skip tests that require the network
      checkFlags =
        let
          e2eTests = [
            "TestValidateUsername/existing_organisation"
            "TestValidateUsername/existing_organisation_with_bad_case"
            "TestValidateUsername/existing_username"
            "TestValidateUsername/existing_username_with_bad_case"
            "TestValidateUsername/non_existing_username"
            "TestValidateProject/public_project"
          ];
        in
        [ "-skip=^${lib.concatStringsSep "$|^" e2eTests}$" ];

      meta = {
        description = "Distributed bug tracker embedded in Git";
        homepage = "https://github.com/git-bug/git-bug";
        license = lib.licenses.gpl3Plus;
      };
    };
in
{
  # build target for the current system
  default = buildTarget { inherit version; };

  # this drv builds release binaries for every platform. the output directory
  # contains all of the binaries uniquely named with their platform, suitable
  # for uploading as release artifacts.
  release = pkgs.symlinkJoin {
    name = "git-bug-release-binaries";
    paths = map (
      platform:
      buildTarget {
        inherit platform version;
        release = true;
      }
    ) (builtins.attrNames releaseTargets);
  };
}
