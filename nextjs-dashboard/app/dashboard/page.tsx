import type { Metadata } from 'next';
import { Workspace } from '@/app/components/workspace';
export const metadata: Metadata = { title: 'My timesheets' };
export default function DashboardPage() { return <Workspace />; }
