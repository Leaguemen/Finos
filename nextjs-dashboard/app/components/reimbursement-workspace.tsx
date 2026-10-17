'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import Link from 'next/link';
import { ArrowDownTrayIcon, BanknotesIcon, DocumentTextIcon, PlusIcon } from '@heroicons/react/24/outline';
import { WorkspaceShell } from './workspace-shell';
import { CollectionControls } from './collection-controls';
import { Button } from './ui';
import { useAuth } from './auth-provider';
import { bearerHeaders, getTokenRole, SESSION_KEY } from '@/app/lib/auth';
import { filterAndSort } from '@/app/lib/collection';
import { apiRequest, displayDate, localToday, type Employee, type TimesheetOption } from '@/app/lib/timesheets';
import { displayAmount, isReimbursement, reimbursementRequest, reimbursementStatuses, type Reimbursement } from '@/app/lib/reimbursements';

const sorts = [
  { value: 'date-desc', label: 'Date · newest first' },
  { value: 'date-asc', label: 'Date · oldest first' },
  { value: 'amount-desc', label: 'Amount · highest first' },
  { value: 'amount-asc', label: 'Amount · lowest first' },
  { value: 'category-asc', label: 'Category · A to Z' },
  { value: 'category-desc', label: 'Category · Z to A' },
];

function ReimbursementList() {
  const { session } = useAuth();
  const isManager = getTokenRole(session?.token) === 'manager';
  const [rows, setRows] = useState<Reimbursement[]>([]);
  const [timesheets, setTimesheets] = useState<TimesheetOption[]>([]);
  const [employees, setEmployees] = useState<Employee[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [reload, setReload] = useState(0);
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState('');
  const [category, setCategory] = useState('');
  const [timesheetDescription, setTimesheetDescription] = useState('');
  const [userId, setUserId] = useState('');
  const [receipt, setReceipt] = useState('');
  const [from, setFrom] = useState('');
  const [to, setTo] = useState('');
  const [sort, setSort] = useState('date-desc');
  const [exporting, setExporting] = useState(false);
  const [exportError, setExportError] = useState('');
  const [receiptError, setReceiptError] = useState('');
  const [openingReceipt, setOpeningReceipt] = useState<number | null>(null);
  const exportInProgress = useRef(false);

  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    setError('');
    async function load() {
      const [timesheetData, employeeData] = await Promise.all([
        apiRequest<{ timesheets: TimesheetOption[] | null }>('/api/timesheets/options', { signal: controller.signal }),
        isManager ? apiRequest<{ users: Employee[] | null }>('/api/users', { signal: controller.signal }) : Promise.resolve({ users: [] }),
      ]);
      const nextTimesheets = timesheetData.timesheets ?? [];
      const nextEmployees = employeeData.users ?? [];
      if (!Array.isArray(nextTimesheets) || !Array.isArray(nextEmployees)) throw new Error('The server returned an invalid list.');
      if (nextTimesheets.some(option => !Number.isInteger(option.id) || option.id <= 0 || (option.description !== null && typeof option.description !== 'string'))) throw new Error('The server returned an invalid timesheet option list.');
      const lists = await Promise.all(nextTimesheets.map(timesheet => reimbursementRequest<{ reimbursements: Reimbursement[] | null }>(`/api/timesheets/${timesheet.id}/reimbursements`, { signal: controller.signal })));
      const nextRows = lists.flatMap(list => list.reimbursements ?? []);
      if (lists.some(list => list.reimbursements !== null && !Array.isArray(list.reimbursements)) || !nextRows.every(isReimbursement)) throw new Error('The server returned an invalid reimbursement list.');
      if (nextEmployees.some(employee => !Number.isInteger(employee.id) || employee.id <= 0 || typeof employee.name !== 'string' || !employee.name.trim())) throw new Error('The server returned an invalid employee list.');
      if (!controller.signal.aborted) { setTimesheets(nextTimesheets); setEmployees(nextEmployees); setRows(nextRows); }
    }
    load().catch(reason => { if (!controller.signal.aborted) setError(reason instanceof Error ? reason.message : 'Unable to load reimbursements.'); })
      .finally(() => { if (!controller.signal.aborted) setLoading(false); });
    return () => controller.abort();
  }, [isManager, reload]);

  const employeeNames = useMemo(() => new Map(employees.map(employee => [employee.id, employee.name.trim()])), [employees]);
  const timesheetMap = useMemo(() => new Map(timesheets.map(timesheet => [timesheet.id, timesheet])), [timesheets]);
  const employeeName = (id: number) => employeeNames.get(id) ?? (id === session?.user?.id ? session.user.name : 'Unknown employee');
  const timesheetDescriptionFor = (id: number) => timesheetMap.get(id)?.description?.trim() || 'No description';
  const categories = [...new Set(rows.map(row => row.category.trim()))].sort((a, b) => a.localeCompare(b));
  const timesheetDescriptions = [...new Set(rows.map(row => timesheetMap.get(row.timesheet_id)?.description?.trim() || null))]
    .sort((left, right) => left === null ? 1 : right === null ? -1 : left.localeCompare(right));
  const rowUserIds = new Set(rows.map(row => row.user_id));
  const filterEmployees = employees.filter(employee => rowUserIds.has(employee.id)).sort((a, b) => a.name.localeCompare(b.name));
  const invalidRange = !!from && !!to && from > to;
  const query = search.trim().toLowerCase();
  const filtered = filterAndSort(rows, [
    row => !query || row.category.toLowerCase().includes(query) || (row.description ?? '').toLowerCase().includes(query) || timesheetDescriptionFor(row.timesheet_id).toLowerCase().includes(query),
    row => !status || row.status === status,
    row => !category || row.category === category,
    row => !timesheetDescription || JSON.stringify(timesheetMap.get(row.timesheet_id)?.description?.trim() || null) === timesheetDescription,
    row => !isManager || !userId || String(row.user_id) === userId,
    row => !receipt || (receipt === 'with' ? row.has_receipt : !row.has_receipt),
    row => !from || row.expense_date >= from,
    row => !to || row.expense_date <= to,
  ], row => sort.startsWith('amount') ? Number(row.amount) : sort.startsWith('category') ? row.category.toLowerCase() : row.expense_date, sort.endsWith('asc') ? 'asc' : 'desc');

  const reset = () => { setSearch(''); setStatus(''); setCategory(''); setTimesheetDescription(''); setUserId(''); setReceipt(''); setFrom(''); setTo(''); setSort('date-desc'); };

  async function exportRows() {
    if (exportInProgress.current || loading || error || invalidRange || !filtered.length) return;
    exportInProgress.current = true; setExporting(true); setExportError('');
    try {
      const { downloadReimbursements } = await import('@/app/lib/reimbursement-export');
      await downloadReimbursements(filtered, localToday(), employeeName, timesheetMap, isManager);
    } catch { setExportError('Unable to export your reimbursements. Please try again.'); }
    finally { exportInProgress.current = false; setExporting(false); }
  }

  async function openReceipt(id: number) {
    if (openingReceipt !== null) return;
    const preview = window.open('', '_blank');
    if (preview) preview.opener = null;
    setOpeningReceipt(id); setReceiptError('');
    try {
      const response = await fetch(`/api/reimbursements/${id}/receipt`, { headers: bearerHeaders(), cache: 'no-store', signal: AbortSignal.timeout(30_000) });
      if (response.status === 401) { localStorage.removeItem(SESSION_KEY); window.dispatchEvent(new Event('storage')); }
      if (!response.ok) { const body = await response.json().catch(() => null); throw new Error(body?.error?.message || 'Unable to open this receipt.'); }
      const url = URL.createObjectURL(await response.blob());
      if (preview) preview.location.href = url; else window.location.href = url;
      window.setTimeout(() => URL.revokeObjectURL(url), 60_000);
    } catch (reason) { preview?.close(); setReceiptError(reason instanceof Error ? reason.message : 'Unable to open this receipt.'); }
    finally { setOpeningReceipt(null); }
  }

  return <>
    <p className="eyebrow"><span />YOUR WORKSPACE</p>
    <div className="workspace-title"><div><h1>{isManager ? 'Team reimbursements.' : 'My reimbursements.'}</h1><p>{isManager ? 'A clear ledger of your team’s work expenses.' : 'Your work expenses, receipts, and status in one place.'}</p></div><div className="workspace-actions">
      <Button type="button" className="button-secondary" onClick={exportRows} disabled={loading || !!error || invalidRange || !filtered.length || exporting} aria-busy={exporting} title="Export reimbursements matching the current filters">{exporting ? 'Exporting…' : 'Export to Excel'}<ArrowDownTrayIcon aria-hidden="true" /></Button>
      <Link className="finos-button" href="/reimbursements/new">New reimbursement<PlusIcon /></Link>
    </div></div>
    {exportError && <p className="form-error page-message" role="alert">{exportError}</p>}
    {receiptError && <p className="form-error page-message" role="alert">{receiptError}</p>}
    <section className="timesheet-panel reimbursement-panel" aria-label={isManager ? 'Team reimbursements' : 'Your reimbursements'}>
      <CollectionControls search={search} onSearch={setSearch} sort={sort} onSort={setSort} options={sorts} onReset={reset} searchPlaceholder="Search categories or descriptions…">
        {isManager && <label>Employee<select value={userId} onChange={event => setUserId(event.target.value)}><option value="">All employees</option>{filterEmployees.map(employee => <option key={employee.id} value={employee.id}>{employee.name}</option>)}</select></label>}
        <label>Timesheet<select value={timesheetDescription} onChange={event => setTimesheetDescription(event.target.value)}><option value="">All timesheets</option>{timesheetDescriptions.map(description => <option key={JSON.stringify(description)} value={JSON.stringify(description)}>{description || 'No description'}</option>)}</select></label>
        <label>Category<select value={category} onChange={event => setCategory(event.target.value)}><option value="">All categories</option>{categories.map(value => <option key={value} value={value}>{value}</option>)}</select></label>
        <label>Status<select value={status} onChange={event => setStatus(event.target.value)}><option value="">All statuses</option>{reimbursementStatuses.map(value => <option key={value} value={value}>{value[0].toUpperCase() + value.slice(1)}</option>)}</select></label>
        <label>Receipt<select value={receipt} onChange={event => setReceipt(event.target.value)}><option value="">All records</option><option value="with">With receipt</option><option value="without">Without receipt</option></select></label>
        <label>From<input type="date" value={from} onChange={event => setFrom(event.target.value)} /></label>
        <label>To<input type="date" value={to} onChange={event => setTo(event.target.value)} /></label>
      </CollectionControls>
      {invalidRange && <p className="form-error inline-message" role="alert">The end date must be on or after the start date.</p>}
      {loading ? <div className="list-state" role="status">Loading reimbursements…</div> : error ? <div className="list-state"><p role="alert">{error}</p><Button className="button-secondary" onClick={() => setReload(value => value + 1)}>Try again</Button></div> : <>
        <div className="list-summary"><span>{filtered.length} of {rows.length} reimbursements</span><strong>{displayAmount(filtered.reduce((sum, row) => sum + Number(row.amount), 0).toFixed(2))} total shown</strong></div>
        {!filtered.length ? <div className="list-state"><BanknotesIcon /><h2>{rows.length ? 'No matching reimbursements.' : 'Your expense ledger is ready.'}</h2><p>{rows.length ? 'Try another search or adjust your filters.' : 'Create a reimbursement and attach a receipt when you have one.'}</p>{rows.length ? <Button className="button-secondary" onClick={reset}>Clear filters</Button> : <Link className="finos-button" href="/reimbursements/new">Create a reimbursement<PlusIcon /></Link>}</div> : <div className="timesheet-table-wrap"><table className="timesheet-table reimbursement-table"><thead><tr><th>Expense date</th>{isManager && <th>Employee</th>}<th>Expense</th><th>Timesheet</th><th className="amount-column">Amount</th><th>Status</th><th>Receipt</th></tr></thead><tbody>{filtered.map(row => {
          return <tr key={row.id}><td data-label="Expense date"><time dateTime={row.expense_date}>{displayDate(row.expense_date)}</time></td>{isManager && <td data-label="Employee">{employeeName(row.user_id)}</td>}
            <td data-label="Expense" className="description-cell"><strong className="category-name">{row.category}</strong>{row.description ? <span className="expense-description">{row.description}</span> : <span className="muted expense-description">No description</span>}</td>
            <td data-label="Timesheet" className="timesheet-description-cell">{timesheetDescriptionFor(row.timesheet_id)}</td><td data-label="Amount" className="amount-column">{displayAmount(row.amount)}</td>
            <td data-label="Status"><span className={`status-pill status-${row.status}`}>{row.status}</span></td><td data-label="Receipt">{row.has_receipt ? <Button className="receipt-button button-secondary" onClick={() => openReceipt(row.id)} disabled={openingReceipt !== null} aria-busy={openingReceipt === row.id}>{openingReceipt === row.id ? 'Opening…' : 'View'}<DocumentTextIcon /></Button> : <span className="muted">None</span>}</td></tr>;
        })}</tbody></table></div>}
      </>}
    </section>
  </>;
}

export function ReimbursementWorkspace() {
  const { session } = useAuth();
  return <WorkspaceShell><ReimbursementList key={session?.token} /></WorkspaceShell>;
}
