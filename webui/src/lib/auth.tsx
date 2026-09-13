// auth.tsx — current user hook for the webui.
//
// The UserIdentity query is preloaded in the root route loader and consumed
// via useSuspenseQuery, so useAuth() always returns a resolved user.

import { useSuspenseQuery } from "@apollo/client/react";

import { graphql } from "@/__generated__/gql";

export const USER_IDENTITY_QUERY = graphql(`
  query UserIdentity {
    repository {
      name
      userIdentity {
        ...IdentitySummary
        id
        humanId
        displayName
        avatarUrl
        name
        email
        login
      }
    }
  }
`);

export function useAuth() {
  const { data } = useSuspenseQuery(USER_IDENTITY_QUERY);
  return { user: data.repository?.userIdentity ?? null };
}
