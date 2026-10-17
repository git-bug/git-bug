import type { ResultOf } from "@graphql-typed-document-node/core";
import { createFileRoute } from "@tanstack/react-router";

import { graphql } from "@/__generated__/gql";

export const REFS_QUERY = graphql(`
  query CodePageRefs($repo: String) {
    repository(ref: $repo) {
      name
      head {
        shortName
      }
      refs {
        ...RefSelectorRefs
      }
    }
  }
`);

export type RefsQueryData = ResultOf<typeof REFS_QUERY>;

export const Route = createFileRoute("/$repo")({
  beforeLoad: ({ params: { repo }, context: { preloadQuery } }) => {
    // Normalize the repo slug: "_" means the default repo, selected by a null ref
    const ref = repo === "_" ? null : repo;

    // Preload refs once for the entire repo — shared by code browser,
    // and used for the /$repo → tree redirect.
    const refsRef = preloadQuery(REFS_QUERY, {
      variables: { repo: ref },
    });

    return { ref, refsRef };
  },
});
