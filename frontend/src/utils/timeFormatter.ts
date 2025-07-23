/**
 * Formats seconds into a human-readable time string
 * @param seconds - Time in seconds
 * @returns Formatted time string (e.g., "2h 30m 15s" or "45m 20s")
 */
export function formatTime(seconds: number): string {
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const secs = Math.floor(seconds % 60);

  if (hours > 0) {
    return `${hours}h ${minutes}m ${secs}s`;
  }
  return `${minutes}m ${secs}s`;
}
