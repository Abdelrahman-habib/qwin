import { Monitor, Settings, ChartColumnBig } from "lucide-react";

export const sidebarList = [
  {
    id: "dashboard",
    name: "Dashboard",
    icon: Monitor,
    path: "/",
  },
  {
    id: "app-usage",
    name: "App Usage",
    icon: ChartColumnBig,
    path: "/usage",
  },
  {
    id: "settings",
    name: "Settings",
    icon: Settings,
    path: "/settings",
  },
] as const;
