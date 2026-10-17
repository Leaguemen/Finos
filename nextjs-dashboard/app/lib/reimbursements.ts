import { bearerHeaders, SESSION_KEY } from './auth';

export type ReimbursementStatus = 'draft' | 'submitted' | 'approved' | 'rejected';

export type Reimbursement = {
  id: number;
  user_id: number;
  timesheet_id: number;
  expense_date: string;
  amount: string;
  category: string;
  description: string | null;
  status: ReimbursementStatus;
  has_receipt: boolean;
  receipt_url?: string | null;
  created_at: string;
  updated_at: string;
};

export const reimbursementStatuses: ReimbursementStatus[] = ['draft', 'submitted', 'approved', 'rejected'];

export async function reimbursementRequest<T>(path: string, options: RequestInit = {}): Promise<T> {
  const response = await fetch(path, {
    ...options,
    cache: 'no-store',
    headers: { ...bearerHeaders(), ...options.headers },
    signal: options.signal ?? AbortSignal.timeout(30_000),
  });
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

export function displayAmount(amount: string) {
  const value = Number(amount);
  return Number.isFinite(value)
    ? value.toLocaleString('en-GB', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
    : amount;
}

export function isReimbursement(value: unknown): value is Reimbursement {
  if (!value || typeof value !== 'object') return false;
  const item = value as Partial<Reimbursement>;
  return Number.isInteger(item.id) && Number(item.id) > 0 &&
    Number.isInteger(item.user_id) && Number(item.user_id) > 0 &&
    Number.isInteger(item.timesheet_id) && Number(item.timesheet_id) > 0 &&
    typeof item.expense_date === 'string' && /^\d{4}-\d{2}-\d{2}$/.test(item.expense_date) &&
    typeof item.amount === 'string' && /^\d{1,10}(?:\.\d{2})$/.test(item.amount) && Number(item.amount) > 0 &&
    typeof item.category === 'string' && !!item.category.trim() &&
    (item.description === null || typeof item.description === 'string') &&
    reimbursementStatuses.includes(item.status as ReimbursementStatus) &&
    typeof item.has_receipt === 'boolean';
}
