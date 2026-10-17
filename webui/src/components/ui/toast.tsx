// Toasts: short notices in the top-right corner of the screen, below the
// header, like the changes others make to the bug being viewed.  <Toasts> is
// mounted once by the Shell; any code shows a toast with toastManager.add(),
// giving an id to replace a previous toast rather than stack a new one.

import { Toast } from "@base-ui/react/toast";
import { X } from "lucide-react";
import type { ReactNode } from "react";

import { buttonVariants } from "@/components/ui/button";

export const toastManager = Toast.createToastManager();

export function Toasts({ children }: { children: ReactNode }) {
  return (
    <Toast.Provider toastManager={toastManager}>
      {children}
      <Toast.Portal>
        <Toast.Viewport className="fixed top-18 right-4 z-50 flex w-80 flex-col gap-2">
          <ToastList />
        </Toast.Viewport>
      </Toast.Portal>
    </Toast.Provider>
  );
}

function ToastList() {
  const { toasts } = Toast.useToastManager();
  return toasts.map((toast) => (
    <Toast.Root
      key={toast.id}
      toast={toast}
      className="bg-popover text-popover-foreground rounded-lg border p-3 shadow-lg transition-all duration-200 data-ending-style:-translate-y-2 data-ending-style:opacity-0 data-starting-style:-translate-y-2 data-starting-style:opacity-0"
    >
      <Toast.Content className="flex items-start gap-3">
        <div className="min-w-0 flex-1">
          <Toast.Title className="text-sm font-medium" />
          <Toast.Description className="text-muted-foreground mt-0.5 text-xs" />
        </div>
        <Toast.Action className={buttonVariants({ size: "sm", variant: "outline" })} />
        <Toast.Close
          aria-label="Close"
          className="text-muted-foreground hover:text-foreground rounded-sm p-0.5"
        >
          <X className="size-4" />
        </Toast.Close>
      </Toast.Content>
    </Toast.Root>
  ));
}
