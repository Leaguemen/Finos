import type { Metadata } from 'next';
import { AuthShell } from '@/app/components/auth-shell';
import { AuthForm } from '@/app/components/auth-form';
export const metadata: Metadata = { title: 'Create an account' };
export default function RegisterPage() { return <AuthShell><AuthForm mode="register" /></AuthShell>; }
