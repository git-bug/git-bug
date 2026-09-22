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
        "border-border flex items-start gap-3 border-b px-4 py-3.5 last:border-0",
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
  // leading-6 rather than the inherited 1.5: titles wrap often in this list, and
  // the extra leading is what makes a two-line title readable at a glance.
  return <div className="flex flex-wrap items-baseline gap-2 leading-6">{children}</div>;
}

interface MetaProps {
  children: React.ReactNode;
}

export function Meta({ children }: MetaProps) {
  // 13px, not text-xs: this line carries the id, age and author of every row,
  // so it is read as much as the title itself.
  return <p className="text-muted-foreground mt-1 text-[13px]">{children}</p>;
}

interface CommentCountProps {
  count: number;
}

export function CommentCount({ count }: CommentCountProps) {
  if (count <= 0) return null;
  return (
    <div className="text-muted-foreground flex shrink-0 items-center gap-1 text-[13px]">
      <MessageSquare className="size-3.5" />
      {count}
    </div>
  );
}
