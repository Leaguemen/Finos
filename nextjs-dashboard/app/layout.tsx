import '@/app/ui/global.css';
import type { Metadata } from 'next';
import { AuthProvider } from '@/app/components/auth-provider';
export const metadata: Metadata = {
  title: { default: 'Finos', template: '%s | Finos' },
  description: 'Your work, beautifully organized. Welcome to Finos.',
};
export default function RootLayout({ children }: { children: React.ReactNode }) {
  return <html lang="en"><body><AuthProvider>{children}</AuthProvider></body></html>;
}
