import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts";
import {
  ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";
import { types } from "../../../wailsjs/go/models";
import { APP_CONFIG } from "../../constants/app";
import { AppIcon } from "./AppIcon";
import { LoadingSkeleton } from "./LoadingSkeleton";
import { EmptyState } from "./EmptyState";
import { JSX } from "react/jsx-runtime";

interface AppUsageChartProps {
  apps: types.AppUsage[];
  isLoading: boolean;
}

const chartConfig = {
  AppUsage: {
    label: "minutes",
    color: "#2563eb",
  },
} satisfies ChartConfig;

// Custom Tick component for X-axis to display app icons
interface CustomXAxisTickProps {
  x: number;
  y: number;
  payload: { value: string };
  appIcons: Record<string, string | undefined>;
}

const CustomXAxisTick = (props: CustomXAxisTickProps) => {
  const { x, y, payload, appIcons } = props;
  const appName = payload.value;
  const iconPath = appIcons[appName];

  return (
    <foreignObject x={x - 15} y={y} width={30} height={30}>
      <AppIcon appName={appName} iconPath={iconPath} className="size-6" />
    </foreignObject>
  );
};

export function AppUsageChart({ apps, isLoading }: AppUsageChartProps) {
  if (isLoading) {
    return (
      <div className="p-3 h-full flex items-center justify-center">
        <LoadingSkeleton />
      </div>
    );
  }

  if (apps.length === 0) {
    return (
      <div className="p-3 h-full flex items-center justify-center">
        <EmptyState />
      </div>
    );
  }

  const chartData = apps.slice(0, APP_CONFIG.MAX_APPS_DISPLAY).map((app) => ({
    appName: app.name,
    AppUsage: Math.round(app.duration / 60), // Convert seconds to minutes
    iconPath: app.iconPath, // Store icon path for custom tick
  }));

  const appIcons = chartData.reduce((acc, curr) => {
    acc[curr.appName] = curr.iconPath;
    return acc;
  }, {} as Record<string, string | undefined>);

  return (
    <div className="p-3 h-full">
      <div className="text-xs text-muted-foreground mb-2">Most Used Apps</div>
      <ChartContainer
        config={chartConfig}
        className="h-[calc(100%-30px)] w-full"
      >
        <BarChart
          accessibilityLayer
          data={chartData}
          margin={{ top: 5, right: 0, left: 0, bottom: 5 }}
        >
          <CartesianGrid
            vertical={false}
            stroke="#374151"
            strokeOpacity={0.3}
          />
          <XAxis
            dataKey="appName"
            tickLine={false}
            axisLine={false}
            interval={0} // Ensure all ticks are displayed
            tick={(props: JSX.IntrinsicAttributes & CustomXAxisTickProps) => (
              <CustomXAxisTick {...props} appIcons={appIcons} />
            )}
            height={40} // Adjust height to accommodate icons
          />
          <YAxis
            dataKey="AppUsage"
            tickLine={false}
            axisLine={false}
            tickFormatter={(value: string) => `${value}m`}
            domain={[0, "dataMax + 10"]}
            width={30}
          />
          <ChartTooltip content={<ChartTooltipContent />} />
          <Bar dataKey="AppUsage" fill="var(--color-AppUsage)" radius={6} />
        </BarChart>
      </ChartContainer>
    </div>
  );
}
