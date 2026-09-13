import { useMutation } from "@apollo/client/react";
import { Pencil } from "lucide-react";
import { useState, useRef, useEffect } from "react";

import { graphql } from "@/__generated__/gql";
import { BugDetailDocument } from "@/__generated__/graphql";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useAuth } from "@/lib/auth";

const BUG_SET_TITLE_MUTATION = graphql(`
  mutation BugSetTitle($input: BugSetTitleInput!) {
    bugSetTitle(input: $input) {
      bug {
        id
        title
      }
    }
  }
`);

interface TitleEditorProps {
  bugPrefix: string;
  title: string;
  humanId: string;
  /** Current repo slug, passed as `ref` in refetch query variables. */
  ref_?: string | null;
}

// Inline title editor in BugDetailPage. Shows the title as plain text with a
// pencil icon on hover (auth-gated). Enter saves, Escape cancels.
export function TitleEditor({ bugPrefix, title, humanId, ref_ }: TitleEditorProps) {
  const { user } = useAuth();
  const [editing, setEditing] = useState(false);
  const [value, setValue] = useState(title);
  const inputRef = useRef<HTMLInputElement>(null);

  const [setTitle, { loading }] = useMutation(BUG_SET_TITLE_MUTATION, {
    refetchQueries: [{ query: BugDetailDocument, variables: { ref: ref_, prefix: bugPrefix } }],
  });

  useEffect(() => {
    if (editing) inputRef.current?.focus();
  }, [editing]);

  async function handleSave() {
    const trimmed = value.trim();
    if (trimmed && trimmed !== title) {
      await setTitle({
        variables: { input: { prefix: bugPrefix, title: trimmed, repoRef: ref_ } },
      });
    }
    setEditing(false);
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === "Enter") void handleSave();
    if (e.key === "Escape") {
      setValue(title);
      setEditing(false);
    }
  }

  if (editing) {
    return (
      <div className="flex items-start gap-2">
        <Input
          ref={inputRef}
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={handleKeyDown}
          className="text-xl font-semibold"
          disabled={loading}
        />
        <Button
          size="sm"
          onClick={() => {
            void handleSave();
          }}
          disabled={loading || !value.trim()}
        >
          Save
        </Button>
        <Button
          size="sm"
          variant="ghost"
          onClick={() => {
            setValue(title);
            setEditing(false);
          }}
        >
          Cancel
        </Button>
      </div>
    );
  }

  return (
    <div className="group flex items-start gap-2">
      <h1 className="text-foreground flex-1 text-2xl leading-tight font-semibold">
        {title}
        <span className="text-muted-foreground ml-2 text-xl font-normal">#{humanId}</span>
      </h1>
      {user && (
        <button
          onClick={() => {
            // Seed the draft from the current title, which may have changed
            // since the last edit (e.g. after a refetch).
            setValue(title);
            setEditing(true);
          }}
          className="text-muted-foreground hover:text-foreground mt-1 shrink-0 opacity-0 transition-opacity group-hover:opacity-100"
          title="Edit title"
        >
          <Pencil className="size-4" />
        </button>
      )}
    </div>
  );
}
