export function dateLabel(
  value: string,
  options: Intl.DateTimeFormatOptions = {
    month: "short",
    day: "numeric",
    year: "numeric",
  },
) {
  return new Intl.DateTimeFormat("en", { ...options, timeZone: "UTC" }).format(
    new Date(`${value}T12:00:00Z`),
  );
}
export function tripStatus(start: string, end: string, timezone: string) {
  const today = new Intl.DateTimeFormat("en-CA", {
    timeZone: timezone,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(new Date());
  return today < start ? "Upcoming" : today > end ? "Past" : "In progress";
}
export function crossesMidnight(start: string | null, duration: number) {
  if (!start) return false;
  const [h, m] = start.split(":").map(Number);
  return h * 60 + m + duration > 1440;
}
