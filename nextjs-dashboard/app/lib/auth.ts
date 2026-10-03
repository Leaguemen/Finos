export type User = { id: number; name: string; email: string; role: string };
export type AuthResponse = { access_token: string; token_type: string; expires_in: number; user: User };
export type Session = { token: string; expiresAt: number; user?: User };
export const SESSION_KEY = 'finos.session';
export function readSession(): Session | null {
  try {
    const value = localStorage.getItem(SESSION_KEY);
    if (!value) return null;
    const session: Session = JSON.parse(value);
    if (typeof session.token !== 'string' || !session.token.trim() ||
        !Number.isFinite(session.expiresAt) || session.expiresAt <= Date.now()) {
      localStorage.removeItem(SESSION_KEY);
      return null;
    }
    return session;
  } catch { return null; }
}
// Reusable for future protected API calls. The backend must validate the token.
export function bearerHeaders(): HeadersInit {
  const session = readSession();
  return session ? { Authorization: `Bearer ${session.token}` } : {};
}
export async function authenticate(mode: 'login' | 'register', input: {
  email: string; password: string; name?: string;
}): Promise<AuthResponse> {
  const response = await fetch(`/api/auth/${mode}`, {
    method: 'POST', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input), signal: AbortSignal.timeout(20_000),
  });
  const data = await response.json().catch(() => null);
  if (!response.ok) throw new Error(data?.error?.message || (mode === 'login'
    ? 'Unable to log in. Please check your email and password.'
    : 'Unable to create your account. Please try again.'));
  if (typeof data?.access_token !== 'string' || !data.access_token.trim() ||
      typeof data.expires_in !== 'number' || !Number.isFinite(data.expires_in) || data.expires_in <= 0 ||
      data.token_type?.toLowerCase() !== 'bearer') {
    throw new Error('The server did not return a valid session. Please try logging in.');
  }
  return data;
}
