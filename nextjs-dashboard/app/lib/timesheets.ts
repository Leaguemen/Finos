import { bearerHeaders, SESSION_KEY } from './auth';
export const statuses = ['draft', 'submitted', 'approved', 'rejected'] as const;
export type Timesheet = {
  id: number; user_id: number; work_date: string; hours_worked: number;
  description: string | null; status: typeof statuses[number]; created_at: string; updated_at: string;
};
export type CreateTimesheet = Pick<Timesheet, 'work_date' | 'hours_worked' | 'description'>;
export async function apiRequest<T>(path: string, options: RequestInit = {}): Promise<T> {
  const response = await fetch(path, { ...options, cache: 'no-store', headers: {
    'Content-Type': 'application/json', ...bearerHeaders(), ...options.headers,
  }, signal: options.signal ?? AbortSignal.timeout(20_000) });
  if (response.status === 401) {
    localStorage.removeItem(SESSION_KEY);
    window.dispatchEvent(new Event('storage'));
    throw new Error('Your session has expired. Please log in again.');
  }
  const body = await response.json().catch(() => null);
  if (!response.ok) throw new Error(body?.error?.message || 'Unable to complete your request. Please try again.');
  if (!body) throw new Error('The server returned an invalid response. Please try again.');
  return body;
}
export function displayDate(date: string) {
  return new Date(`${date}T12:00:00`).toLocaleDateString('en-GB', { day: 'numeric', month: 'short', year: 'numeric' });
}
export function localToday() {
  const now = new Date();
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`;
}
