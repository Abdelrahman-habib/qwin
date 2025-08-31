import { Settings, ChartColumnBig, CalendarCheck } from "lucide-react";

export const sidebarList = [
  {
    id: "palnner",
    name: "Palnner",
    icon: CalendarCheck,
    path: "/planner",
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
