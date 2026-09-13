import { createRootRouteWithContext, useRouter } from "@tanstack/react-router";
import { AlertTriangle } from "lucide-react";

import { Shell } from "@/components/layout/shell";
import { Button } from "@/components/ui/button";
import { ButtonLink } from "@/components/ui/button-link";
import type { preloadQuery } from "@/lib/apollo";
import { USER_IDENTITY_QUERY } from "@/lib/auth";

export interface RouterContext {
  preloadQuery: typeof preloadQuery;
}

export const Route = createRootRouteWithContext<RouterContext>()({
  async loader({ context }) {
    const ref = context.preloadQuery(USER_IDENTITY_QUERY);
    await context.preloadQuery.toPromise(ref);
  },
  component: Shell,
  errorComponent: ErrorPage,
});

function ErrorPage({ error }: { error?: unknown }) {
  const router = useRouter();

  const message = error instanceof Error ? error.message : "An unexpected error occurred.";

  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-4 text-center">
      <AlertTriangle className="text-muted-foreground size-10" />
      <p className="text-muted-foreground text-sm">{message}</p>
      <div className="flex gap-2">
        <Button
          variant="outline"
          size="sm"
          onClick={() => {
            void router.invalidate();
          }}
        >
          Try again
        </Button>
        <ButtonLink to="/" variant="outline" size="sm">
          Go home
        </ButtonLink>
      </div>
    </div>
  );
}
