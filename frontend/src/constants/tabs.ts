import { LearningTracker } from "@/components/learning-tracker";
import { ScreenTimeWidget } from "@/components/ScreenTimeWidget";
import { Tab } from "@/types/tabs";
import { BookCheckIcon, TimerIcon } from "lucide-react";

export const tabs: Tab[] = [
  {
    id: "usage",
    name: "App Usage",
    icon: TimerIcon,
    content: ScreenTimeWidget,
  },
  {
    id: "learning",
    name: "Learning",
    icon: BookCheckIcon,
    content: LearningTracker,
  },
];
