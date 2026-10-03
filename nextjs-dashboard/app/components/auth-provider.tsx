'use client';
import { createContext, useContext, useEffect, useState, type ReactNode } from 'react';
import { useRouter } from 'next/navigation';
import { readSession, SESSION_KEY, type AuthResponse, type Session } from '@/app/lib/auth';
const AuthContext = createContext<{
  session: Session | null; ready: boolean;
  signIn: (response: AuthResponse) => void; signOut: () => void;
} | null>(null);
export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<Session | null>(null);
  const [ready, setReady] = useState(false);
  useEffect(() => {
    const sync = () => { setSession(readSession()); setReady(true); };
    sync();
    window.addEventListener('storage', sync);
    window.addEventListener('focus', sync);
    const timer = window.setInterval(sync, 30_000);
    return () => {
      window.removeEventListener('storage', sync);
      window.removeEventListener('focus', sync);
      window.clearInterval(timer);
    };
  }, []);
  useEffect(() => {
    if (!session) return;
    const timer = window.setTimeout(() => setSession(readSession()),
      Math.min(Math.max(session.expiresAt - Date.now(), 0), 2_147_483_647));
    return () => window.clearTimeout(timer);
  }, [session]);
  function signIn(response: AuthResponse) {
    const next = { token: response.access_token, expiresAt: Date.now() + response.expires_in * 1000, user: response.user };
    try { localStorage.setItem(SESSION_KEY, JSON.stringify(next)); }
    catch { throw new Error('Please allow browser storage to stay signed in to Finos.'); }
    setSession(next);
  }
  function signOut() {
    try { localStorage.removeItem(SESSION_KEY); } finally { setSession(null); }
  }
  return <AuthContext.Provider value={{ session, ready, signIn, signOut }}>{children}</AuthContext.Provider>;
}
export function useAuth() {
  const auth = useContext(AuthContext);
  if (!auth) throw new Error('useAuth requires AuthProvider');
  return auth;
}
export function AuthGate({ children, guest = false }: { children: ReactNode; guest?: boolean }) {
  const { session, ready } = useAuth();
  const router = useRouter();
  const allowed = guest ? !session : !!session;
  useEffect(() => {
    if (ready && !allowed) router.replace(guest ? '/dashboard' : '/login');
  }, [ready, allowed, guest, router]);
  if (!ready || !allowed) return <div className="session-loading" role="status"><span className="loading-dot" />Opening Finos…</div>;
  return children;
}
