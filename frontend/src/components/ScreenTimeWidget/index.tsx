import { useScreenTime } from "../../hooks/useScreenTime";
import { APP_CONFIG } from "../../constants/app";
import { TotalTimeDisplay } from "./TotalTimeDisplay";
import { AppUsageChart } from "./AppUsageChart";
import { ErrorState } from "./ErrorState";
import { useTitle } from "@/hooks/use-title";

export function ScreenTimeWidget() {
  const { setTitle } = useTitle();

  const { usageData, isLoading, error } = useScreenTime(
    APP_CONFIG.REFRESH_INTERVAL
  );

  if (error) {
    return <ErrorState error={error} />;
  }

  setTitle("Total Today");

  return (
    <>
      <TotalTimeDisplay totalTime={usageData.totalTime} isLoading={isLoading} />
      <AppUsageChart apps={usageData.apps} isLoading={isLoading} />
    </>
  );
}
