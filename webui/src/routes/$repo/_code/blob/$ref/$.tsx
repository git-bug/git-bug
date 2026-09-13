// Blob (file) view: /$repo/blob/$ref/...path

import { useReadQuery } from "@apollo/client/react";
import { createFileRoute } from "@tanstack/react-router";

import { graphql } from "@/__generated__/gql";
import { FileContent } from "@/components/code/file-content";
import { Skeleton } from "@/components/ui/skeleton";

const BLOB_QUERY = graphql(`
  query CodePageBlob($repo: String, $ref: String!, $path: String!) {
    repository(ref: $repo) {
      name
      blob(ref: $ref, path: $path) {
        ...FileViewerBlob
      }
    }
  }
`);

function BlobSkeleton() {
  return (
    <div className="border-border overflow-hidden rounded-md border">
      <div className="flex items-center gap-2 border-b px-4 py-2">
        <Skeleton className="h-4 w-48" />
      </div>
      <div className="p-4">
        <Skeleton className="h-64 w-full" />
      </div>
    </div>
  );
}

export const Route = createFileRoute("/$repo/_code/blob/$ref/$")({
  component: BlobView,
  pendingComponent: BlobSkeleton,
  beforeLoad: () => ({ viewMode: "blob" as const }),
  loader: async ({ context: { preloadQuery, ref }, params: { ref: gitRef, _splat: path } }) => {
    const blobRef = preloadQuery(BLOB_QUERY, {
      variables: { repo: ref, ref: gitRef, path: path || "" },
    });
    return { blobRef: await preloadQuery.toPromise(blobRef) };
  },
});

function BlobView() {
  const { repo, ref: gitRef } = Route.useParams();
  const { blobRef } = Route.useLoaderData();
  const { data } = useReadQuery(blobRef);

  const blob = data?.repository?.blob;
  if (!blob) return <BlobSkeleton />;
  return <FileContent blob={blob} repo={repo} gitRef={gitRef} />;
}
