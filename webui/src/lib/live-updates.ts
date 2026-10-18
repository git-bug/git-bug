// Live updates: a view subscribes to the entities it displays, and refetches
// its query when the server says they changed, wherever the change came from
// (another tab, the terminal UI, a pull).  The server only says what changed,
// never the new state, so the refetch goes through the usual queries.
//
// The cache only stays current for what a view subscribes to, so the views
// load with fetchPolicy "network-only": what they showed earlier may be stale.

import { type QueryRef, useApolloClient, useSubscription } from "@apollo/client/react";
import type { DocumentNode } from "graphql";
import { useCallback, useState } from "react";

import type { EntityType } from "@/__generated__/enums";
import { graphql } from "@/__generated__/gql";

const ENTITIES_CHANGED_SUBSCRIPTION = graphql(`
  subscription EntitiesChanged($ref: String, $type: EntityType!, $ids: [ID!]) {
    entitiesChanged(repoRef: $ref, type: $type, ids: $ids) {
      id
    }
  }
`);

// Calls onChange with the ids of the entities of the type that changed, all of
// them or only those given by ids, at most once per second.
export function useEntitiesChanged(
  ref: string | null,
  type: EntityType,
  ids: string[] | null,
  onChange: (ids: string[]) => void,
) {
  useSubscription(ENTITIES_CHANGED_SUBSCRIPTION, {
    variables: { ref, type, ids },
    skip: ids?.length === 0,
    ignoreResults: true,
    onData: ({ data }) => {
      onChange(data.data?.entitiesChanged.map((change) => change.id) ?? []);
    },
  });
}

// Collects the entities of the type that changed since the view loaded queryRef
// or last refreshed, for views like lists that offer a refresh rather than
// moving under the user.  refresh refetches the active queries of the document
// and clears the collected changes, restoring them if the refetch fails.
export function usePendingChanges(
  ref: string | null,
  type: EntityType,
  query: DocumentNode,
  queryRef: QueryRef<unknown, never>,
) {
  // the changes belong to the queryRef they were collected for, so that a
  // refresh failing after the view loaded another one doesn't restore them
  const [pending, setPending] = useState<{
    queryRef: QueryRef<unknown, never>;
    ids: ReadonlySet<string>;
  }>({ queryRef, ids: new Set() });
  if (pending.queryRef !== queryRef) {
    setPending({ queryRef, ids: new Set() });
  }
  useEntitiesChanged(ref, type, null, (ids) => {
    setPending((prev) => ({ ...prev, ids: new Set([...prev.ids, ...ids]) }));
  });

  const client = useApolloClient();
  const refresh = useCallback(() => {
    // cleared now rather than on success, to keep the changes arriving meanwhile
    const refreshed = pending;
    setPending({ ...refreshed, ids: new Set() });
    client
      .refetchQueries({ include: [query] })
      .catch(() =>
        setPending((prev) =>
          prev.queryRef === refreshed.queryRef
            ? { ...prev, ids: new Set([...refreshed.ids, ...prev.ids]) }
            : prev,
        ),
      );
  }, [client, pending, query]);

  return { count: pending.ids.size, refresh };
}

// Returns a function refetching the active queries of the document, with their
// own variables.  The results land in the cache, which updates every view
// reading them, whichever component holds the query ref, and the views keep
// showing their current data meanwhile.
export function useRefetch(query: DocumentNode) {
  const client = useApolloClient();
  return useCallback(() => {
    // on failure the view keeps its data, and the next change refetches again
    client.refetchQueries({ include: [query] }).catch(() => {});
  }, [client, query]);
}
