import { Outlet } from "@tanstack/react-router";

import { Header } from "./header";

// Top-level page wrapper used as the root layout in App.tsx. Renders the
// Header above the current route's page component via <Outlet>.
export function Shell() {
  return (
    <div className="bg-background min-h-screen font-sans">
      <Header />
      <main className="mx-auto max-w-screen-xl px-4 py-6">
        <Outlet />
      </main>
    </div>
  );
}
