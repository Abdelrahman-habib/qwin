import { createFileRoute } from "@tanstack/react-router";
import { ScreenTimeWidget } from "@/features/apps-usage/components";

export const Route = createFileRoute("/usage")({
  component: RouteComponent,
});

function RouteComponent() {
  return (
    <div className="p-6 animate-in slide-in-from-bottom-4 fade-in duration-300">
      <ScreenTimeWidget />
    </div>
  );
}
