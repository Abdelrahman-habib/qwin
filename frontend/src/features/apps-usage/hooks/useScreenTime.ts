import { useState, useEffect } from "react";
import { GetUsageData } from "@wailsjs/go/app/App";
import { types } from "@wailsjs/go/models";

/**
 * Custom hook for managing screen time data
 * @param refreshInterval - Refresh interval in milliseconds (default: 5000)
 * @returns Object containing usage data, loading state, and error state
 */
export function useScreenTime(refreshInterval: number = 5000) {
  const [usageData, setUsageData] = useState<types.UsageData>(
    new types.UsageData({ totalTime: 0, apps: [] })
  );
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadUsageData = async () => {
    try {
      setError(null);
      const data = await GetUsageData();
      setUsageData(data);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to load usage data"
      );
      console.error("Failed to load usage data:", err);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    // Initial load
    loadUsageData();

    // Set up interval for periodic updates
    const interval = setInterval(loadUsageData, refreshInterval);

    return () => clearInterval(interval);
  }, [refreshInterval]);

  return {
    usageData,
    isLoading,
    error,
    refresh: loadUsageData,
  };
}
