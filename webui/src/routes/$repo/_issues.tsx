import { createFileRoute, Outlet } from "@tanstack/react-router";

import { EntityType } from "@/__generated__/enums";
import { graphql } from "@/__generated__/gql";
import { useEntitiesChanged, useRefetch } from "@/lib/live-updates";

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

const REPO_LABELS_QUERY = graphql(`
  query RepoLabels($ref: String) {
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
      knownLabels {
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
  component: IssuesLayout,
  beforeLoad: ({ context: { preloadQuery, ref } }) => {
    const labelsRef = preloadQuery(REPO_LABELS_QUERY, {
      variables: { ref },
    });
    const identitiesRef = preloadQuery(ALL_IDENTITIES_QUERY, {
      variables: { ref },
    });
    return { labelsRef, identitiesRef };
  },
});

// Keeps the shared identities and labels current for the pages reading them,
// like the author filter and the label pickers: a refetch only reaches a query
// a page reads.  The labels change with the bugs using them, and with the config
// defining and allowing them.
function IssuesLayout() {
  const { ref } = Route.useRouteContext();
  useEntitiesChanged(ref, EntityType.Identity, null, useRefetch(ALL_IDENTITIES_QUERY));
  const refetchLabels = useRefetch(REPO_LABELS_QUERY);
  useEntitiesChanged(ref, EntityType.Bug, null, refetchLabels);
  useEntitiesChanged(ref, EntityType.Config, null, refetchLabels);
  return <Outlet />;
}
