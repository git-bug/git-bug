import { Outlet } from "@tanstack/react-router";

import { Toasts } from "@/components/ui/toast";

import { Header } from "./header";

// Top-level page wrapper used as the root layout in App.tsx. Renders the
// Header above the current route's page component via <Outlet>, and the
// toasts of every page.
export function Shell() {
  return (
    <Toasts>
      <div className="bg-background min-h-screen font-sans">
        <Header />
        <main className="mx-auto max-w-screen-xl px-4 py-6">
          <Outlet />
        </main>
      </div>
    </Toasts>
  );
}
