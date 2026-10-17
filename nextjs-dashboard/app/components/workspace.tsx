'use client';
import { useEffect, useRef, useState } from 'react';
import Link from 'next/link';
import { PlusIcon, ClockIcon, ArrowDownTrayIcon } from '@heroicons/react/24/outline';
import { WorkspaceShell } from './workspace-shell';
import { CollectionControls } from './collection-controls';
import { Button } from './ui';
import { useAuth } from './auth-provider';
import { getTokenRole } from '@/app/lib/auth';
import { apiRequest, displayDate, localToday, statuses, type Employee, type Timesheet } from '@/app/lib/timesheets';
import { filterAndSort } from '@/app/lib/collection';
const sorts = [
  { value: 'date-desc', label: 'Date · newest first' }, { value: 'date-asc', label: 'Date · oldest first' },
  { value: 'hours-desc', label: 'Hours · highest first' }, { value: 'hours-asc', label: 'Hours · lowest first' },
];
function TimesheetList() {
  const { session } = useAuth();
  const isManager = getTokenRole(session?.token) === 'manager';
  const [userId, setUserId] = useState('');
  const [approving, setApproving] = useState<number[]>([]);
  const [approvalErrors, setApprovalErrors] = useState<Record<number, string>>({});
  const [approvalNotice, setApprovalNotice] = useState('');
  const approvalsInProgress = useRef(new Set<number>());
  const [rows, setRows] = useState<Timesheet[]>([]);
  const [employees, setEmployees] = useState<Employee[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [reload, setReload] = useState(0);
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState('');
  const [from, setFrom] = useState('');
  const [to, setTo] = useState('');
  const [sort, setSort] = useState('date-desc');
  const [exporting, setExporting] = useState(false);
  const [exportError, setExportError] = useState('');
  const exportInProgress = useRef(false);
  useEffect(() => {
    const controller = new AbortController();
    setLoading(true); setError('');
    const timesheetsRequest = apiRequest<{ timesheets: Timesheet[] | null }>('/api/timesheets', { signal: controller.signal });
    const employeesRequest = isManager
      ? apiRequest<{ users: Employee[] | null }>('/api/users', { signal: controller.signal })
      : Promise.resolve({ users: [] });

    Promise.all([timesheetsRequest, employeesRequest])
      .then(([timesheetData, employeeData]) => {
        if (controller.signal.aborted) return;
        if (timesheetData.timesheets !== null && !Array.isArray(timesheetData.timesheets)) {
          throw new Error('The server returned an invalid timesheet list.');
        }
        if (employeeData.users !== null && !Array.isArray(employeeData.users)) {
          throw new Error('The server returned an invalid employee list.');
        }
        const nextEmployees = employeeData.users ?? [];
        if (nextEmployees.some(employee => !Number.isInteger(employee.id) || employee.id <= 0 || typeof employee.name !== 'string' || !employee.name.trim())) {
          throw new Error('The server returned an invalid employee list.');
        }
        setRows(timesheetData.timesheets ?? []);
        setEmployees(nextEmployees);
      })
      .catch(error => { if (!controller.signal.aborted) setError(error instanceof Error ? error.message : 'Unable to load timesheets.'); })
      .finally(() => { if (!controller.signal.aborted) setLoading(false); });
    return () => controller.abort();
  }, [isManager, reload]);
  const invalidRange = !!from && !!to && from > to;
  const employeeNames = new Map(employees.map(employee => [employee.id, employee.name.trim()]));
  const timesheetUserIds = new Set(rows.map(row => row.user_id));
  const filterEmployees = employees
    .filter(employee => timesheetUserIds.has(employee.id))
    .sort((left, right) => left.name.localeCompare(right.name));
  const employeeName = (id: number) => employeeNames.get(id) ?? 'Unknown employee';
  const filtered = filterAndSort(rows, [
    row => (row.description ?? '').toLowerCase().includes(search.trim().toLowerCase()),
    row => !status || row.status === status,
    row => !isManager || !userId || String(row.user_id) === userId,
    row => !from || row.work_date >= from, row => !to || row.work_date <= to,
  ], row => sort.startsWith('hours') ? row.hours_worked : row.work_date, sort.endsWith('asc') ? 'asc' : 'desc');
  const reset = () => { setSearch(''); setStatus(''); setUserId(''); setFrom(''); setTo(''); setSort('date-desc'); };
  async function approveTimesheet(row: Timesheet) {
    if (!isManager || approvalsInProgress.current.has(row.id)) return;
    approvalsInProgress.current.add(row.id);
    setApproving(ids => [...ids, row.id]);
    setApprovalErrors(errors => ({ ...errors, [row.id]: '' }));
    setApprovalNotice('');
    try {
      const updated = await apiRequest<Timesheet>(`/api/timesheets/${row.id}/approve`, { method: 'PATCH' });
      if (updated.id !== row.id || !['approved', 'draft'].includes(updated.status)) throw new Error('The server did not confirm the status change. Please refresh and try again.');
      setRows(current => current.map(item => item.id === updated.id ? updated : item));
      setApprovalNotice(`Timesheet #${row.id} for ${employeeName(row.user_id)} ${updated.status === 'approved' ? 'approved' : 'returned to draft'}.`);
    } catch (error) {
      setApprovalErrors(errors => ({ ...errors, [row.id]: error instanceof Error ? error.message : 'Unable to update this timesheet. Please try again.' }));
    } finally {
      approvalsInProgress.current.delete(row.id);
      setApproving(ids => ids.filter(id => id !== row.id));
    }
  }
  async function exportTimesheets() {
    if (exportInProgress.current || loading || error || invalidRange || !filtered.length) return;
    exportInProgress.current = true;
    setExporting(true);
    setExportError('');
    try {
      // Load the Excel writer only when the user requests an export.
      const { downloadTimesheets } = await import('@/app/lib/timesheet-export');
      await downloadTimesheets(filtered, localToday());
    } catch {
      setExportError('Unable to export your timesheets. Please try again.');
    } finally {
      exportInProgress.current = false;
      setExporting(false);
    }
  }
  return <>
    <p className="eyebrow"><span />YOUR WORKSPACE</p>
    <div className="workspace-title"><div><h1>{isManager ? 'Team timesheets.' : 'My timesheets.'}</h1><p>{isManager ? 'Review your team’s work and approve their timesheets.' : 'A little record of the work that moves you forward.'}</p></div><div className="workspace-actions">
      <Button type="button" className="button-secondary" onClick={exportTimesheets}
        disabled={loading || !!error || invalidRange || !filtered.length || exporting}
        aria-busy={exporting} title="Export all timesheets matching the current search and filters">
        {exporting ? 'Exporting…' : 'Export to Excel'}<ArrowDownTrayIcon aria-hidden="true" />
      </Button>
      <Link className="finos-button" href="/timesheets/new">New timesheet<PlusIcon /></Link>
    </div></div>
    {exportError && <p className="form-error" role="alert">{exportError}</p>}
    <p className="approval-notice" role="status">{approvalNotice}</p>
    <section className="timesheet-panel" aria-label={isManager ? 'Team timesheets' : 'Your timesheets'}>
      <CollectionControls search={search} onSearch={setSearch} sort={sort} onSort={setSort} options={sorts} onReset={reset}>
        {isManager && <label>Employee<select value={userId} onChange={e => setUserId(e.target.value)}><option value="">All employees</option>{filterEmployees.map(employee => <option key={employee.id} value={employee.id}>{employee.name}</option>)}</select></label>}
        <label>Status<select value={status} onChange={e => setStatus(e.target.value)}><option value="">All statuses</option>{statuses.map(s => <option key={s} value={s}>{s[0].toUpperCase() + s.slice(1)}</option>)}</select></label>
        <label>From<input type="date" value={from} onChange={e => setFrom(e.target.value)} /></label>
        <label>To<input type="date" value={to} onChange={e => setTo(e.target.value)} /></label>
      </CollectionControls>
      {invalidRange && <p className="form-error" role="alert">The end date must be on or after the start date.</p>}
      {loading ? <div className="list-state" role="status">Loading your timesheets…</div> : error ? <div className="list-state"><p role="alert">{error}</p><Button className="button-secondary" onClick={() => setReload(n => n + 1)}>Try again</Button></div> : <>
        <div className="list-summary" role="status"><span>{filtered.length} of {rows.length} timesheets</span><strong>{filtered.reduce((sum, row) => sum + row.hours_worked, 0)} hours shown</strong></div>
        {filtered.length === 0 ? <div className="list-state"><ClockIcon /><h2>{rows.length ? 'No matching timesheets.' : isManager ? 'No team timesheets yet.' : 'Your first entry starts here.'}</h2><p>{rows.length ? 'Try another search or adjust your filters.' : 'Keep track of your working day, one timesheet at a time.'}</p>{rows.length ? <Button className="button-secondary" onClick={reset}>Clear filters</Button> : <Link className="finos-button" href="/timesheets/new">Create a timesheet<PlusIcon /></Link>}</div> : <div className="timesheet-table-wrap"><table className="timesheet-table"><thead><tr><th>Work date</th>{isManager && <th>Employee</th>}<th>Description</th><th>Hours</th><th>Status</th>{isManager && <th className="approval-column">Approval</th>}</tr></thead><tbody>{filtered.map(row => <tr key={row.id}>
          <td data-label="Work date"><time dateTime={row.work_date}>{displayDate(row.work_date)}</time></td>
          {isManager && <td data-label="Employee">{employeeName(row.user_id)}</td>}
          <td data-label="Description" className="description-cell">{row.description || <span className="muted">No description</span>}</td>
          <td data-label="Hours">{row.hours_worked}h</td><td data-label="Status"><span className={`status-pill status-${row.status}`}>{row.status}</span></td>
          {isManager && <td className="approval-column" data-label="Approval"><div className="approval-action">
            <Button type="button" className="button-secondary" disabled={approving.includes(row.id)}
              aria-label={`${row.status === 'approved' ? 'Return to draft:' : 'Approve'} timesheet #${row.id} for ${employeeName(row.user_id)}`} aria-busy={approving.includes(row.id)}
              onClick={() => approveTimesheet(row)}>{approving.includes(row.id) ? (row.status === 'approved' ? 'Returning to draft…' : 'Approving…') : row.status === 'approved' ? 'Return to draft' : 'Approve'}</Button>
            {approvalErrors[row.id] && <p className="form-error" role="alert">{approvalErrors[row.id]}</p>}
          </div></td>}
        </tr>)}</tbody></table></div>}
      </>}
    </section>
  </>;
}
export function Workspace() {
  const { session } = useAuth();
  return <WorkspaceShell><TimesheetList key={session?.token} /></WorkspaceShell>;
}
