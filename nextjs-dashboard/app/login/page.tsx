import type { Metadata } from 'next';
import { AuthShell } from '@/app/components/auth-shell';
import { AuthForm } from '@/app/components/auth-form';
export const metadata: Metadata = { title: 'Log in' };
export default function LoginPage() { return <AuthShell><AuthForm mode="login" /></AuthShell>; }
