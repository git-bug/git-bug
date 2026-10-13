// Commit history view: /$repo/commits/$ref?path=...

import { createFileRoute } from "@tanstack/react-router";
import * as v from "valibot";

import { CommitList } from "@/components/code/commit-list";

const commitsSearchSchema = v.object({
  path: v.fallback(v.optional(v.string()), undefined),
});

export const Route = createFileRoute("/$repo/_code/commits/$ref")({
  validateSearch: (search) => v.parse(commitsSearchSchema, search),
  beforeLoad: () => ({ viewMode: "commits" as const }),
  component: CommitsView,
});

function CommitsView() {
  const { ref: currentRef } = Route.useParams();
  const { ref: repoRef } = Route.useRouteContext();
  const { path } = Route.useSearch();

  return <CommitList repo={repoRef} gitRef={currentRef} path={path} />;
}
