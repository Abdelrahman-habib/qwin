export interface AppUsage {
  name: string;
  duration: number;
}

export interface UsageData {
  totalTime: number;
  apps: AppUsage[];
}
