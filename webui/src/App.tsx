import { createRouter, RouterProvider } from "@tanstack/react-router";

import { preloadQuery } from "@/lib/apollo";

import { routeTree } from "./routeTree.gen";

const router = createRouter({
  routeTree,
  defaultPreload: "intent",
  // Apollo owns the data: run the loaders on every navigation, so a preload
  // never stands in for the fetch on arrival
  defaultPreloadStaleTime: 0,
  scrollRestoration: true,
  context: { preloadQuery },
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}

export function App() {
  return <RouterProvider router={router} />;
}
