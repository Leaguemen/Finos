'use client';
import type { ReactNode } from 'react';
export function CollectionControls({ search, onSearch, sort, onSort, options, children, onReset }: {
  search: string; onSearch: (value: string) => void; sort: string; onSort: (value: string) => void;
  options: { value: string; label: string }[]; children?: ReactNode; onReset: () => void;
}) {
  return <div className="collection-controls">
    <label className="search-control">Search<input type="search" value={search} onChange={e => onSearch(e.target.value)} placeholder="Search descriptions…" /></label>
    {children}
    <label>Sort by<select value={sort} onChange={e => onSort(e.target.value)}>{options.map(option => <option key={option.value} value={option.value}>{option.label}</option>)}</select></label>
    <button type="button" className="reset-filters" onClick={onReset}>Reset filters</button>
  </div>;
}
