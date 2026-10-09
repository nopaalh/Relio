export const SNAPSHOT_DATE = "2026-10-01";
export function validDate(value: string): boolean {
  return /^\d{4}-\d{2}-\d{2}$/.test(value) && !Number.isNaN(Date.parse(value)) &&
    new Date(value).toISOString().slice(0,10) === value;
}
export function assertDate(value: string) {
  if (!validDate(value) || value > SNAPSHOT_DATE || value < "2024-01-01")
    throw new Error("Select a valid date between 1 January 2024 and 1 October 2026.");
}
export function visibleAt(date: string | null | undefined, asOf: string): boolean {
  return Boolean(date && validDate(date) && date <= asOf);
}
export function attractiveness(acv: number | null, max: number, outlets: number | null): number | null {
  if (acv === null || outlets === null || max <= 0) return null;
  return Math.round(.5 * (100 * acv / max) + .5 * (outlets > 30 ? 100 : 0));
}
export function validateSelection(ids: string[], available: string[]): string | null {
  if (ids.length < 2 || ids.length > 4) return "Select 2–4 historical approaches.";
  if (new Set(ids).size !== ids.length) return "Approaches must be unique.";
  if (ids.some(id => !available.includes(id))) return "Approach unknown or not sourced.";
  return null;
}
export function fingerprint(deal: string, date: string, ids: string[], version: string): string {
  return [deal,date,[...ids].sort().join(","),version].join("|");
}
export function scoreLabel(value: number | null): string { return value === null ? "Unavailable" : String(value); }
export function validChoice(values: (number | null)[]): boolean {
  return values.length >= 2 && values.every(v => v !== null && Number.isFinite(v) && v >= 0 && v <= 1)
    && Math.abs(values.reduce<number>((sum,v) => sum + (v ?? 0),0) - 1) <= .02;
}
