import Link from 'next/link';
export function Brand() {
  return <Link href="/" className="brand" aria-label="Finos home">
    <span className="brand-mark" aria-hidden="true"><svg viewBox="0 0 32 32" fill="none"><path d="M9 24V8h15M9 16h11" stroke="currentColor" strokeWidth="3.5"/><path d="m20 23 5-5" stroke="currentColor" strokeWidth="3.5"/></svg></span>
    finos<span className="brand-period">.</span>
  </Link>;
}
