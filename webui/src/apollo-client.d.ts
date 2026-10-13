import type { GraphQLCodegenDataMasking } from "@apollo/client/masking";

declare module "@apollo/client" {
  interface TypeOverrides extends GraphQLCodegenDataMasking.TypeOverrides {}
}
