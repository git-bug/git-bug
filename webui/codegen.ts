import type { CodegenConfig } from "@graphql-codegen/cli";

const config: CodegenConfig = {
  overwrite: true,
  schema: "../api/graphql/schema/*.graphql",
  documents: ["src/**/*.{ts,tsx}", "!src/__generated__/**/*"],
  ignoreNoDocuments: true,
  generates: {
    "./src/__generated__/enums.ts": {
      plugins: ["typescript"],
      config: {
        onlyEnums: true,
        enumsAsConst: true,
      },
    },
    "./src/__generated__/": {
      preset: "client",
      presetConfig: {
        fragmentMasking: false,
      },
      config: {
        useTypeImports: true,
        avoidOptionals: {
          field: true,
          inputValue: false,
        },
        defaultScalarType: "unknown",
        nonOptionalTypename: true,
        skipTypeNameForRoot: true,
        // Generate masked inline fragment types for Apollo's data masking
        inlineFragmentTypes: "mask",
        customDirectives: {
          apolloUnmask: true,
        },
        scalars: {
          Time: "string",
          Hash: "string",
          CombinedId: "string",
          Color: "{ R: number; G: number; B: number }",
        },
      },
    },
  },
};

export default config;
