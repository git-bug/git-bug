// Row at the top of an issue list telling that issues changed since the list
// was loaded, with a button to refresh it.  Used by the issue list and the user
// profile pages, which offer the refresh rather than moving under the user.

import { RefreshCw } from "lucide-react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface ChangesBannerProps {
  count: number;
  onRefresh: () => void;
  className?: string;
}

export function ChangesBanner({ count, onRefresh, className }: ChangesBannerProps) {
  if (count === 0) return null;

  return (
    <div
      role="status"
      className={cn(
        "border-border bg-muted/50 flex items-center justify-between gap-2 border-b px-4 py-2 text-sm",
        className,
      )}
    >
      <span className="text-muted-foreground">
        {count === 1 ? "1 issue changed" : `${count} issues changed`}
      </span>
      <Button variant="outline" size="sm" onClick={onRefresh}>
        <RefreshCw data-icon="inline-start" />
        Refresh
      </Button>
    </div>
  );
}
