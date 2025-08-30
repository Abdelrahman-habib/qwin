import { APP_CONFIG } from "@/constants/app";

import { useScreenTime } from "../hooks/useScreenTime";

import { TotalTimeDisplay } from "./TotalTimeDisplay";
import { AppUsageChart } from "./AppUsageChart";
import { ErrorState } from "./ErrorState";

export function ScreenTimeWidget() {
  const { usageData, isLoading, error } = useScreenTime(
    APP_CONFIG.REFRESH_INTERVAL
  );

  if (error) {
    return <ErrorState error={error} />;
  }

  return (
    <>
      <TotalTimeDisplay totalTime={usageData.totalTime} isLoading={isLoading} />
      <AppUsageChart apps={usageData.apps} isLoading={isLoading} />
    </>
  );
}
