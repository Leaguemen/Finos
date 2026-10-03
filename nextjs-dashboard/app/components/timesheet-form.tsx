'use client';
import { useRef, useState, type FormEvent } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { ArrowLeftIcon, CheckIcon } from '@heroicons/react/24/outline';
import { WorkspaceShell } from './workspace-shell';
import { Field, Button } from './ui';
import { apiRequest, localToday, type Timesheet } from '@/app/lib/timesheets';
function Form() {
  const [pending, setPending] = useState(false);
  const [error, setError] = useState('');
  const submitting = useRef(false);
  const router = useRouter();
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (submitting.current) return;
    const data = new FormData(event.currentTarget);
    const hours = Number(data.get('hours_worked'));
    if (!Number.isInteger(hours) || hours < 1 || hours > 24) { setError('Enter a whole number of hours between 1 and 24.'); return; }
    submitting.current = true; setPending(true); setError('');
    try {
      await apiRequest<Timesheet>('/api/timesheets', { method: 'POST', body: JSON.stringify({
        work_date: String(data.get('work_date')), hours_worked: hours, description: String(data.get('description') || '').trim() || null,
      }) });
      router.push('/dashboard');
    } catch (error) {
      setError(error instanceof Error ? error.message : 'Unable to save your timesheet. Please try again.');
      submitting.current = false; setPending(false);
    }
  }
  return <>
    <Link className="back-link" href="/dashboard"><ArrowLeftIcon />My timesheets</Link>
    <div className="workspace-title"><div><p className="eyebrow">MAKE YOUR WORK COUNT</p><h1>New timesheet.</h1><p>A quick note of your day. A clearer picture of your work.</p></div></div>
    <div className="create-layout"><form className="timesheet-form" onSubmit={submit} aria-label="New timesheet" aria-busy={pending}>
      <fieldset disabled={pending}><div className="form-row">
        <Field id="work-date" name="work_date" label="Work date" type="date" defaultValue={localToday()} required />
        <Field id="hours-worked" name="hours_worked" label="Hours worked" type="number" min={1} max={24} step={1} placeholder="8" required hint="Whole hours, from 1 to 24." />
      </div>
      <div className="field"><label htmlFor="description">Description <span className="muted">(optional)</span></label><textarea id="description" name="description" rows={5} placeholder="What did you work on?" /></div>
      {error && <p className="form-error" role="alert">{error}</p>}
      <div className="form-actions"><Link className="back-link" href="/dashboard">Cancel</Link><Button type="submit" disabled={pending}>{pending ? 'Saving…' : 'Save timesheet'}<CheckIcon /></Button></div>
      </fieldset></form>
      <aside className="timesheet-note"><p className="eyebrow">ONE DAY AT A TIME</p><h2>Small entries.<br />A clearer picture.</h2><p>Your timesheet will be saved as a <strong>draft</strong>. You can find it in My timesheets after saving.</p><p>Submitting and editing timesheets will be available later.</p></aside>
    </div>
  </>;
}
export function TimesheetForm() { return <WorkspaceShell><Form /></WorkspaceShell>; }
