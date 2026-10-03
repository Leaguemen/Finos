'use client';
import { useState, type FormEvent } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { ArrowRightIcon, LockClosedIcon } from '@heroicons/react/24/outline';
import { authenticate } from '@/app/lib/auth';
import { useAuth } from './auth-provider';
import { Button, Field } from './ui';
export function AuthForm({ mode }: { mode: 'login' | 'register' }) {
  const register = mode === 'register';
  const [error, setError] = useState('');
  const [pending, setPending] = useState(false);
  const { signIn } = useAuth();
  const router = useRouter();
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (pending) return;
    setError('');
    const form = new FormData(event.currentTarget);
    const email = String(form.get('email')).trim();
    const password = String(form.get('password'));
    const name = String(form.get('name') || '').trim();
    if (register && !name) { setError('Please enter your full name.'); return; }
    const bytes = new TextEncoder().encode(password).length;
    if (register && (bytes < 8 || bytes > 72)) {
      setError('Use a password between 8 and 72 bytes. Some special characters use more than one byte.'); return;
    }
    setPending(true);
    try {
      signIn(await authenticate(mode, { email, password, ...(register ? { name } : {}) }));
      router.replace('/dashboard');
    } catch (error) {
      setError(error instanceof Error && error.name !== 'TimeoutError' && error.name !== 'TypeError'
        ? error.message : 'We could not reach Finos. Please try again in a moment.');
    } finally { setPending(false); }
  }
  return <div className="auth-form-content">
    <nav className="auth-tabs" aria-label="Account access">
      <Link href="/login" aria-current={!register ? 'page' : undefined}>Log in</Link>
      <Link href="/register" aria-current={register ? 'page' : undefined}>Create an account</Link>
    </nav>
    <div className="form-heading"><p className="eyebrow">YOUR WORK LOGS, SIMPLIFIED</p>
      <h2>{register ? 'A fresh start.' : 'Welcome back.'}</h2>
      <p>{register ? 'Create your account and make room for better work.' : 'A new day. A clear start. Log in to your workspace.'}</p>
    </div>
    <form onSubmit={submit} aria-label={register ? 'Create an account' : 'Log in'} aria-busy={pending}>
      <fieldset disabled={pending}>
        {register && <Field id="name" name="name" label="Full name" placeholder="Your full name" autoComplete="name" required maxLength={255} />}
        <Field id="email" name="email" label="Email address" type="email" placeholder="you@company.com" autoComplete="email" autoCapitalize="none" spellCheck={false} required />
        <Field id="password" name="password" label="Password" type="password" placeholder={register ? 'Create a password' : 'Enter your password'} autoComplete={register ? 'new-password' : 'current-password'} required hint={register ? 'Use at least 8 characters. Make it uniquely yours.' : undefined} />
        {error && <div className="form-error" role="alert">{error}</div>}
        <Button type="submit" disabled={pending}>{pending ? (register ? 'Creating your account…' : 'Logging in…') : (register ? 'Create account' : 'Log in')}<ArrowRightIcon aria-hidden="true" /></Button>
      </fieldset>
    </form>
    <p className="switch-auth">{register ? 'Already have an account?' : 'New to Finos?'}{' '}<Link href={register ? '/login' : '/register'}>{register ? 'Log in' : 'Create an account'}<span aria-hidden="true"> ↗</span></Link></p>
    <div className="form-note"><LockClosedIcon aria-hidden="true" /><span>Your workspace. Your own little corner of clarity.</span></div>
  </div>;
}
