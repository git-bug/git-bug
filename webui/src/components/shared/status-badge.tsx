import { CircleDot, CircleCheck } from "lucide-react";

import { Status } from "@/__generated__/enums";
import { cn } from "@/lib/utils";

interface StatusBadgeProps {
  status: Status;
  className?: string;
}

// Open / Closed status badge with icon. Used in BugDetailPage header.
export function StatusBadge({ status, className }: StatusBadgeProps) {
  const isOpen = status === Status.Open;

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium",
        isOpen
          ? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
          : "bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400",
        className,
      )}
    >
      {isOpen ? <CircleDot className="size-3" /> : <CircleCheck className="size-3" />}
      {isOpen ? "Open" : "Closed"}
    </span>
  );
}
