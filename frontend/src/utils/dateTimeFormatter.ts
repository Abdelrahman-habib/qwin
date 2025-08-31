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

/**
 * Formats a date range into a human-readable string
 * @param dateFrom - starting date of the range
 * @param dateTo - ending date of the range
 * @returns Formatted date range string (e.g., "Jun 12, 9am - 10am" or "Jun 12, 9am - Jul 17, 10am")
 */
export function formatDateRange(dateFrom: Date, dateTo: Date): string {
  const options: Intl.DateTimeFormatOptions = {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "numeric",
    hour12: true,
  };

  let from = dateFrom.toLocaleString("en-US", options);
  let to = dateTo.toLocaleString("en-US", options);

  const sameDay = dateFrom.toDateString() === dateTo.toDateString();
  if (sameDay) {
    return `${from} - ${to.slice(12)}`;
  }

  // ensure from is before to else swap
  if (dateFrom > dateTo) {
    [from, to] = [to, from];
  }

  return `${from} - ${to}`;
}
