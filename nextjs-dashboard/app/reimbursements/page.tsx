import type { Metadata } from 'next';
import { ReimbursementWorkspace } from '@/app/components/reimbursement-workspace';

export const metadata: Metadata = { title: 'Reimbursements' };

export default function Page() { return <ReimbursementWorkspace />; }
