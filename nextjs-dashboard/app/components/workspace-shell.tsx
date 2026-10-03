'use client';
import type { ReactNode } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { ArrowRightStartOnRectangleIcon, ClockIcon, BanknotesIcon } from '@heroicons/react/24/outline';
import { AuthGate, useAuth } from './auth-provider';
import { Brand } from './brand';
import { Button } from './ui';
export function WorkspaceShell({ children }: { children: ReactNode }) {
  const { signOut } = useAuth();
  const path = usePathname();
  return <AuthGate><div className="workspace">
    <header className="workspace-header"><Brand /><Button className="button-secondary" onClick={signOut}>Log out<ArrowRightStartOnRectangleIcon /></Button></header>
    <nav className="workspace-nav" aria-label="Workspace">
      <Link href="/dashboard" aria-current={path === '/dashboard' || path.startsWith('/timesheets') ? 'page' : undefined}><ClockIcon />Timesheets</Link>
      <Link href="/reimbursements" aria-current={path === '/reimbursements' ? 'page' : undefined}><BanknotesIcon />Reimbursements</Link>
    </nav>
    <main className="workspace-main">{children}</main>
    <footer className="workspace-footer">© {new Date().getFullYear()} P.T. Doval Sejahtera Investa<span>A little clarity. Every day.</span></footer>
  </div></AuthGate>;
}
