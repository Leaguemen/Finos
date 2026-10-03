'use client';
import { ArrowRightStartOnRectangleIcon, CheckIcon, Squares2X2Icon } from '@heroicons/react/24/outline';
import { AuthGate, useAuth } from './auth-provider';
import { Brand } from './brand';
import { Button } from './ui';
export function Workspace() {
  const { session, signOut } = useAuth();
  const name = session?.user?.name;
  return <AuthGate><div className="workspace">
    <header className="workspace-header"><Brand /><Button className="button-secondary" onClick={signOut}>Log out<ArrowRightStartOnRectangleIcon aria-hidden="true" /></Button></header>
    <main className="workspace-main">
      <p className="eyebrow"><span />YOUR WORKSPACE</p>
      <div className="workspace-title"><div><h1>{name ? `Welcome, ${name.split(' ')[0]}.` : 'Welcome to Finos.'}</h1><p>Good to have you here. Let’s make it a good day.</p></div><span className="session-badge"><span />Logged in</span></div>
      <section className="welcome-card"><span className="welcome-icon"><CheckIcon aria-hidden="true" /></span><p className="eyebrow">YOU’RE ALL SET</p><h2>A clear space.<br />A fresh beginning.</h2><p>You’re logged in to Finos. This is the start of your workspace.<br className="desktop-break" /> More ways to organize your working day are on the way.</p><div className="welcome-card-footer"><Squares2X2Icon aria-hidden="true" />Your workspace starts here</div></section>
      <section className="account-card" aria-label="Your account"><div className="avatar" aria-hidden="true">{name?.charAt(0).toUpperCase() || 'F'}</div><div><h2>{name || 'Your account'}</h2><p>{session?.user?.email || 'You have an active browser session.'}</p></div><span className="account-label">My account</span></section>
    </main><footer className="workspace-footer">© {new Date().getFullYear()} P.T. Doval Sejahtera Investa<span>A little clarity. Every day.</span></footer>
  </div></AuthGate>;
}
