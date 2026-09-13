import { createFileRoute } from "@tanstack/react-router";

import { graphql } from "@/__generated__/gql";

const ALL_IDENTITIES_QUERY = graphql(`
  query AllIdentities($ref: String) {
    repository(ref: $ref) {
      name
      allIdentities(first: 1000) {
        nodes {
          id
          humanId
          name
          email
          login
          displayName
          avatarUrl
        }
      }
    }
  }
`);

const VALID_LABELS_QUERY = graphql(`
  query ValidLabels($ref: String) {
    repository(ref: $ref) {
      name
      validLabels {
        nodes {
          name
          color {
            R
            G
            B
          }
          ...LabelFields
        }
      }
    }
  }
`);

// Pathless layout route for all issue-related pages under /$repo.
// Preloads labels and identities shared by the issue list, detail,
// new issue form, and user profile pages.
export const Route = createFileRoute("/$repo/_issues")({
  beforeLoad: ({ context: { preloadQuery, ref } }) => {
    const labelsRef = preloadQuery(VALID_LABELS_QUERY, {
      variables: { ref },
    });
    const identitiesRef = preloadQuery(ALL_IDENTITIES_QUERY, {
      variables: { ref },
    });
    return { labelsRef, identitiesRef };
  },
});
