import { createRootRoute, Outlet } from "@tanstack/react-router";
import { TanStackRouterDevtools } from "@tanstack/react-router-devtools";

import { ErrorBoundary } from "@/components/ErrorBoundary";
import { ThemeProvider } from "@/components/theme-provider";
import SpotlightCard from "@/components/ui/spotlight-card";
import { Sidebar } from "@/components/navigation/sidebar";
import { RootLayout } from "@/components/layout";
import { ScrollArea } from "@/components/ui/scroll-area";

export const Route = createRootRoute({
  component: () => (
    <ErrorBoundary>
      <ThemeProvider defaultTheme="system" storageKey="vite-ui-theme">
        <SpotlightCard
          className="p-0 bg-transparent"
          spotlightColor="rgba(255, 255, 255, 0.07)"
        >
          <RootLayout>
            <Sidebar />
            <div className="flex-1 border-s border">
              <ScrollArea
                className={
                  "[&>[data-radix-scroll-area-viewport]]:max-h-[calc(100vh_-_3.8rem)]"
                }
              >
                <div className="h-full">
                  <Outlet />
                </div>
              </ScrollArea>
            </div>
          </RootLayout>
        </SpotlightCard>
      </ThemeProvider>
      <TanStackRouterDevtools />
    </ErrorBoundary>
  ),
});
