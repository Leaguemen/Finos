export type SortDirection = 'asc' | 'desc';
export function filterAndSort<T>(items: readonly T[], predicates: ((item: T) => boolean)[], value: (item: T) => string | number, direction: SortDirection): T[] {
  return items.filter(item => predicates.every(predicate => predicate(item))).sort((a, b) => {
    const left = value(a), right = value(b);
    const result = typeof left === 'number' && typeof right === 'number' ? left - right : String(left).localeCompare(String(right));
    return direction === 'asc' ? result : -result;
  });
}
