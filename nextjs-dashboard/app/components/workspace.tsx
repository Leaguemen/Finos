'use client';
import { useEffect, useState } from 'react';
import Link from 'next/link';
import { PlusIcon, ClockIcon } from '@heroicons/react/24/outline';
import { WorkspaceShell } from './workspace-shell';
import { CollectionControls } from './collection-controls';
import { Button } from './ui';
import { apiRequest, displayDate, statuses, type Timesheet } from '@/app/lib/timesheets';
import { filterAndSort } from '@/app/lib/collection';
const sorts = [
  { value: 'date-desc', label: 'Date · newest first' }, { value: 'date-asc', label: 'Date · oldest first' },
  { value: 'hours-desc', label: 'Hours · highest first' }, { value: 'hours-asc', label: 'Hours · lowest first' },
];
function TimesheetList() {
  const [rows, setRows] = useState<Timesheet[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [reload, setReload] = useState(0);
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState('');
  const [from, setFrom] = useState('');
  const [to, setTo] = useState('');
  const [sort, setSort] = useState('date-desc');
  useEffect(() => {
    const controller = new AbortController();
    setLoading(true); setError('');
    apiRequest<{ timesheets: Timesheet[] | null }>('/api/timesheets', { signal: controller.signal })
      .then(data => { if (!controller.signal.aborted) { if (data.timesheets !== null && !Array.isArray(data.timesheets)) throw new Error('The server returned an invalid timesheet list.'); setRows(data.timesheets ?? []); } })
      .catch(error => { if (!controller.signal.aborted) setError(error instanceof Error ? error.message : 'Unable to load timesheets.'); })
      .finally(() => { if (!controller.signal.aborted) setLoading(false); });
    return () => controller.abort();
  }, [reload]);
  const invalidRange = !!from && !!to && from > to;
  const filtered = filterAndSort(rows, [
    row => (row.description ?? '').toLowerCase().includes(search.trim().toLowerCase()),
    row => !status || row.status === status,
    row => !from || row.work_date >= from, row => !to || row.work_date <= to,
  ], row => sort.startsWith('hours') ? row.hours_worked : row.work_date, sort.endsWith('asc') ? 'asc' : 'desc');
  const reset = () => { setSearch(''); setStatus(''); setFrom(''); setTo(''); setSort('date-desc'); };
  return <>
    <p className="eyebrow"><span />YOUR WORKSPACE</p>
    <div className="workspace-title"><div><h1>My timesheets.</h1><p>A little record of the work that moves you forward.</p></div><Link className="finos-button" href="/timesheets/new">New timesheet<PlusIcon /></Link></div>
    <section className="timesheet-panel" aria-label="Your timesheets">
      <CollectionControls search={search} onSearch={setSearch} sort={sort} onSort={setSort} options={sorts} onReset={reset}>
        <label>Status<select value={status} onChange={e => setStatus(e.target.value)}><option value="">All statuses</option>{statuses.map(s => <option key={s} value={s}>{s[0].toUpperCase() + s.slice(1)}</option>)}</select></label>
        <label>From<input type="date" value={from} onChange={e => setFrom(e.target.value)} /></label>
        <label>To<input type="date" value={to} onChange={e => setTo(e.target.value)} /></label>
      </CollectionControls>
      {invalidRange && <p className="form-error" role="alert">The end date must be on or after the start date.</p>}
      {loading ? <div className="list-state" role="status">Loading your timesheets…</div> : error ? <div className="list-state"><p role="alert">{error}</p><Button className="button-secondary" onClick={() => setReload(n => n + 1)}>Try again</Button></div> : <>
        <div className="list-summary" role="status"><span>{filtered.length} of {rows.length} timesheets</span><strong>{filtered.reduce((sum, row) => sum + row.hours_worked, 0)} hours shown</strong></div>
        {filtered.length === 0 ? <div className="list-state"><ClockIcon /><h2>{rows.length ? 'No matching timesheets.' : 'Your first entry starts here.'}</h2><p>{rows.length ? 'Try another search or adjust your filters.' : 'Keep track of your working day, one timesheet at a time.'}</p>{rows.length ? <Button className="button-secondary" onClick={reset}>Clear filters</Button> : <Link className="finos-button" href="/timesheets/new">Create a timesheet<PlusIcon /></Link>}</div> : <div className="timesheet-table-wrap"><table className="timesheet-table"><thead><tr><th>Work date</th><th>Description</th><th>Hours</th><th>Status</th></tr></thead><tbody>{filtered.map(row => <tr key={row.id}>
          <td data-label="Work date"><time dateTime={row.work_date}>{displayDate(row.work_date)}</time></td>
          <td data-label="Description" className="description-cell">{row.description || <span className="muted">No description</span>}</td>
          <td data-label="Hours">{row.hours_worked}h</td><td data-label="Status"><span className={`status-pill status-${row.status}`}>{row.status}</span></td>
        </tr>)}</tbody></table></div>}
      </>}
    </section>
  </>;
}
export function Workspace() { return <WorkspaceShell><TimesheetList /></WorkspaceShell>; }
