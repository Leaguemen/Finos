import type { Metadata } from 'next';
import { TimesheetForm } from '@/app/components/timesheet-form';
export const metadata: Metadata = { title: 'New timesheet' };
export default function Page() { return <TimesheetForm />; }
