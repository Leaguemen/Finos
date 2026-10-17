import type { Metadata } from 'next';
import { ReimbursementForm } from '@/app/components/reimbursement-form';

export const metadata: Metadata = { title: 'New reimbursement' };

export default function Page() { return <ReimbursementForm />; }
