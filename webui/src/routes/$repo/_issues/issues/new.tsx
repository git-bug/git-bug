import { useMutation } from "@apollo/client/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState } from "react";

import { graphql } from "@/__generated__/gql";
import { Markdown } from "@/components/content/markdown";
import * as WritePreview from "@/components/shared/write-preview";
import { BackLink } from "@/components/ui/back-link";
import { Button } from "@/components/ui/button";
import { ButtonLink } from "@/components/ui/button-link";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";

const BUG_CREATE_MUTATION = graphql(`
  mutation BugCreate($input: BugCreateInput!) {
    bugCreate(input: $input) {
      bug {
        id
        humanId
      }
    }
  }
`);

export const Route = createFileRoute("/$repo/_issues/issues/new")({
  component: RouteComponent,
});

// New issue form (/:repo/issues/new). Title + body with write/preview tabs.
function RouteComponent() {
  const navigate = useNavigate();
  const { repo } = Route.useParams();
  const { ref } = Route.useRouteContext();
  const [title, setTitle] = useState("");
  const [message, setMessage] = useState("");
  const [createBug, { loading, error }] = useMutation(BUG_CREATE_MUTATION);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const result = await createBug({
      variables: { input: { title: title.trim(), message: message.trim(), repoRef: ref } },
    });
    const humanId = result.data?.bugCreate.bug.humanId;
    if (humanId) {
      void navigate({
        to: "/$repo/issues/$id",
        params: { repo: repo, id: humanId },
      });
    }
  }

  return (
    <div>
      <BackLink to="/$repo/issues" params={{ repo }} search={{ q: "status:open", after: "" }}>
        Back to issues
      </BackLink>

      <h1 className="mb-6 text-xl font-semibold">New issue</h1>

      <form
        onSubmit={(e) => {
          void handleSubmit(e);
        }}
        className="space-y-4"
      >
        <Input
          placeholder="Title"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          disabled={loading}
          autoFocus
        />

        <WritePreview.Root hasContent={!!message.trim()}>
          <WritePreview.Tabs className="mb-2" />
          <WritePreview.WriteSlot>
            <Textarea
              placeholder="Describe the issue in detail…"
              className="min-h-[200px]"
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              disabled={loading}
            />
          </WritePreview.WriteSlot>
          <WritePreview.PreviewSlot>
            <div className="border-input min-h-[200px] rounded-md border px-3 py-2">
              <Markdown content={message} />
            </div>
          </WritePreview.PreviewSlot>
        </WritePreview.Root>

        {error && (
          <p className="text-destructive text-sm">Failed to create issue: {error.message}</p>
        )}

        <div className="flex justify-end gap-2">
          <ButtonLink
            to="/$repo/issues"
            params={{ repo: repo }}
            search={{ q: "status:open", after: "" }}
            variant="ghost"
          >
            Cancel
          </ButtonLink>
          <Button type="submit" disabled={!title.trim() || loading}>
            {loading ? "Creating…" : "Submit new issue"}
          </Button>
        </div>
      </form>
    </div>
  );
}
