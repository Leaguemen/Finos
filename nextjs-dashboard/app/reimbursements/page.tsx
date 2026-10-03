import type { Metadata } from 'next';
import Link from 'next/link';
import { BanknotesIcon } from '@heroicons/react/24/outline';
import { WorkspaceShell } from '@/app/components/workspace-shell';
export const metadata: Metadata = { title: 'Reimbursements' };
export default function Page() {
  return <WorkspaceShell><p className="eyebrow"><span />YOUR WORKSPACE</p><div className="workspace-title"><div><h1>Reimbursements.</h1><p>A home for your work expenses.</p></div></div><section className="welcome-card reimbursement-placeholder"><span className="welcome-icon"><BanknotesIcon /></span><p className="eyebrow">COMING SOON</p><h2>Less paperwork.<br />More peace of mind.</h2><p>You’ll be able to manage your reimbursement requests here.<br />This space is getting ready. For now, your timesheets are ready to go.</p><Link className="back-link" href="/dashboard">Back to my timesheets →</Link></section></WorkspaceShell>;
}
