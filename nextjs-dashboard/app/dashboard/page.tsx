import type { Metadata } from 'next';
import { Workspace } from '@/app/components/workspace';
export const metadata: Metadata = { title: 'Your workspace' };
export default function DashboardPage() { return <Workspace />; }
