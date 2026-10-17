'use client';

import { useEffect, useRef, useState, type FormEvent } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { ArrowLeftIcon, CheckIcon, PaperClipIcon } from '@heroicons/react/24/outline';
import { WorkspaceShell } from './workspace-shell';
import { Button, Field } from './ui';
import { useAuth } from './auth-provider';
import { getTokenRole } from '@/app/lib/auth';
import { reimbursementRequest, type Reimbursement } from '@/app/lib/reimbursements';
import { apiRequest, localToday, type Timesheet, type TimesheetOption } from '@/app/lib/timesheets';

const MAX_RECEIPT_BYTES = 5 * 1024 * 1024;
const MAX_TIMESHEET_LABEL_LENGTH = 72;

function timesheetLabel(description: string | null) {
  const label = description?.trim() || 'No description';
  return label.length > MAX_TIMESHEET_LABEL_LENGTH ? `${label.slice(0, MAX_TIMESHEET_LABEL_LENGTH - 1).trimEnd()}…` : label;
}

function Form() {
  const { session } = useAuth();
  const isManager = getTokenRole(session?.token) === 'manager';
  const [timesheets, setTimesheets] = useState<TimesheetOption[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState('');
  const [pending, setPending] = useState(false);
  const [error, setError] = useState('');
  const [receiptName, setReceiptName] = useState('No file selected');
  const submitting = useRef(false);
  const router = useRouter();

  useEffect(() => {
    const controller = new AbortController();
    const optionsRequest = apiRequest<{ timesheets: TimesheetOption[] | null }>('/api/timesheets/options', { signal: controller.signal });
    const ownershipRequest = isManager
      ? apiRequest<{ timesheets: Timesheet[] | null }>('/api/timesheets', { signal: controller.signal })
      : Promise.resolve({ timesheets: null });
    Promise.all([optionsRequest, ownershipRequest])
      .then(([optionData, ownershipData]) => {
        if (optionData.timesheets !== null && !Array.isArray(optionData.timesheets)) throw new Error('The server returned an invalid timesheet option list.');
        const options = optionData.timesheets ?? [];
        if (options.some(option => !Number.isInteger(option.id) || option.id <= 0 || (option.description !== null && typeof option.description !== 'string'))) throw new Error('The server returned an invalid timesheet option list.');
        if (!isManager) { setTimesheets(options); return; }
        if (ownershipData.timesheets !== null && !Array.isArray(ownershipData.timesheets)) throw new Error('The server returned an invalid timesheet list.');
        const ownedIds = new Set((ownershipData.timesheets ?? []).filter(row => row.user_id === session?.user?.id).map(row => row.id));
        setTimesheets(options.filter(option => ownedIds.has(option.id)));
      })
      .catch(reason => { if (!controller.signal.aborted) setLoadError(reason instanceof Error ? reason.message : 'Unable to load your timesheets.'); })
      .finally(() => { if (!controller.signal.aborted) setLoading(false); });
    return () => controller.abort();
  }, [isManager, session?.user?.id]);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (submitting.current) return;
    const input = new FormData(event.currentTarget);
    const timesheetId = String(input.get('timesheet_id') ?? '');
    const expenseDate = String(input.get('expense_date') ?? '');
    const amount = String(input.get('amount') ?? '').trim();
    const category = String(input.get('category') ?? '').trim();
    const description = String(input.get('description') ?? '').trim();
    const receipt = input.get('receipt');
    if (!timesheets.some(row => String(row.id) === timesheetId)) { setError('Choose one of your timesheets.'); return; }
    if (!/^\d{4}-\d{2}-\d{2}$/.test(expenseDate)) { setError('Choose a valid expense date.'); return; }
    if (!/^(0|[1-9]\d{0,9})(?:\.\d{1,2})?$/.test(amount) || Number(amount) <= 0) { setError('Enter a positive amount with up to 10 digits and 2 decimal places.'); return; }
    if (!category || category.length > 100) { setError('Enter a category of no more than 100 characters.'); return; }
    if (description.length > 1000) { setError('Keep the description to 1,000 characters or fewer.'); return; }
    if (receipt instanceof File && receipt.size > 0 && (!['image/jpeg', 'image/png'].includes(receipt.type) || receipt.size > MAX_RECEIPT_BYTES)) {
      setError('Choose a JPEG or PNG receipt no larger than 5 MiB.'); return;
    }
    const body = new FormData();
    body.set('expense_date', expenseDate);
    body.set('amount', amount);
    body.set('category', category);
    if (description) body.set('description', description);
    if (receipt instanceof File && receipt.size > 0) body.set('receipt', receipt);
    submitting.current = true; setPending(true); setError('');
    try {
      await reimbursementRequest<Reimbursement>(`/api/timesheets/${timesheetId}/reimbursements`, { method: 'POST', body });
      router.push('/reimbursements');
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Unable to save your reimbursement. Please try again.');
      submitting.current = false; setPending(false);
    }
  }

  return <>
    <Link className="back-link" href="/reimbursements"><ArrowLeftIcon />My reimbursements</Link>
    <div className="workspace-title"><div><p className="eyebrow">RECORD AN EXPENSE</p><h1>New reimbursement.</h1><p>Link the expense to your work, then add the receipt if you have it.</p></div></div>
    <div className="create-layout"><form className="timesheet-form reimbursement-form" onSubmit={submit} aria-label="New reimbursement" aria-busy={pending}>
      <fieldset disabled={pending || loading || !!loadError}>
        <div className="field"><label htmlFor="timesheet-id">Timesheet</label><div className="input-wrap"><select id="timesheet-id" name="timesheet_id" required defaultValue="">
          <option value="" disabled>Choose a timesheet</option>{timesheets.map(row => <option key={row.id} value={row.id} title={row.description?.trim() || 'No description'}>{timesheetLabel(row.description)}</option>)}
        </select></div><p className="field-hint">The expense must belong to one of your timesheets.</p></div>
        <div className="form-row">
          <Field id="expense-date" name="expense_date" label="Expense date" type="date" defaultValue={localToday()} required />
          <Field id="amount" name="amount" label="Amount" type="text" inputMode="decimal" placeholder="125000.00" pattern="(0|[1-9][0-9]{0,9})(\.[0-9]{1,2})?" required hint="Up to 10 digits and 2 decimal places." />
        </div>
        <Field id="category" name="category" label="Category" type="text" maxLength={100} placeholder="e.g. Transportation" required list="expense-categories" />
        <datalist id="expense-categories"><option value="Transportation" /><option value="Meals" /><option value="Accommodation" /><option value="Office supplies" /><option value="Communication" /></datalist>
        <div className="field"><label htmlFor="reimbursement-description">Description <span className="muted">(optional)</span></label><textarea id="reimbursement-description" name="description" rows={5} maxLength={1000} placeholder="What was this expense for?" /></div>
        <div className="field receipt-field"><label htmlFor="receipt">Receipt <span className="muted">(optional)</span></label><input id="receipt" name="receipt" type="file" accept="image/jpeg,image/png" onChange={event => setReceiptName(event.target.files?.[0]?.name ?? 'No file selected')} /><label className="receipt-picker" htmlFor="receipt"><PaperClipIcon /><span>Choose JPEG or PNG</span><small>{receiptName}</small></label><p className="field-hint">Maximum file size: 5 MiB.</p></div>
        {loading && <p className="form-info" role="status">Loading your timesheets…</p>}
        {loadError && <p className="form-error" role="alert">{loadError}</p>}
        {!loading && !loadError && !timesheets.length && <div className="form-empty"><p>You need a timesheet before you can record an expense.</p><Link className="back-link" href="/timesheets/new">Create a timesheet</Link></div>}
        {error && <p className="form-error" role="alert">{error}</p>}
        <div className="form-actions"><Link className="back-link" href="/reimbursements">Cancel</Link><Button type="submit" disabled={pending || loading || !!loadError || !timesheets.length}>{pending ? 'Saving…' : 'Save reimbursement'}<CheckIcon /></Button></div>
      </fieldset>
    </form>
    <aside className="timesheet-note reimbursement-note"><p className="eyebrow">KEEP THE PAPER TRAIL</p><h2>One expense.<br />One clear record.</h2><p>Your reimbursement starts as a <strong>draft</strong>. The amount is stored exactly as entered to preserve decimal precision.</p><p>A receipt is optional. If you add one, Finos accepts JPEG and PNG images up to 5 MiB.</p></aside></div>
  </>;
}

export function ReimbursementForm() { return <WorkspaceShell><Form /></WorkspaceShell>; }
