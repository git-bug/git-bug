import { CircleDot, CircleCheck, MessageSquare } from "lucide-react";

import { Status } from "@/__generated__/enums";
import { graphql } from "@/__generated__/gql";
import { cn } from "@/lib/utils";

graphql(`
  fragment BugSummary on Bug {
    id
    humanId
    status
    title
    labels {
      name
      ...LabelFields
    }
    author {
      ...IdentitySummary
    }
    createdAt
    comments {
      totalCount
    }
  }
`);

interface RootProps {
  className?: string;
  children: React.ReactNode;
}

export function Root({ className, children }: RootProps) {
  return (
    <div
      className={cn(
        "border-border flex items-start gap-3 border-b px-4 py-3 last:border-0",
        className,
      )}
    >
      {children}
    </div>
  );
}

interface StatusIconProps {
  status: Status;
}

export function StatusIcon({ status }: StatusIconProps) {
  const isOpen = status === Status.Open;
  const Icon = isOpen ? CircleDot : CircleCheck;
  return (
    <Icon
      className={cn(
        "mt-0.5 size-4 shrink-0",
        isOpen ? "text-green-600 dark:text-green-400" : "text-purple-600 dark:text-purple-400",
      )}
    />
  );
}

interface TitleAreaProps {
  children: React.ReactNode;
}

export function TitleArea({ children }: TitleAreaProps) {
  return <div className="flex flex-wrap items-baseline gap-2">{children}</div>;
}

interface MetaProps {
  children: React.ReactNode;
}

export function Meta({ children }: MetaProps) {
  return <p className="text-muted-foreground mt-0.5 text-xs">{children}</p>;
}

interface CommentCountProps {
  count: number;
}

export function CommentCount({ count }: CommentCountProps) {
  if (count <= 0) return null;
  return (
    <div className="text-muted-foreground flex shrink-0 items-center gap-1 text-xs">
      <MessageSquare className="size-3.5" />
      {count}
    </div>
  );
}
