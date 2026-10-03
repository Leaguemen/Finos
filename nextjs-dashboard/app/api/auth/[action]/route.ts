import { NextRequest, NextResponse } from 'next/server';
// Same-origin bridge to the Go API avoids browser CORS configuration.
export async function POST(request: NextRequest, context: { params: Promise<{ action: string }> }) {
  const { action } = await context.params;
  if (action !== 'login' && action !== 'register') {
    return NextResponse.json({ error: { message: 'Not found' } }, { status: 404 });
  }
  try {
    const base = (process.env.FINOS_API_URL || 'http://localhost:8080').replace(/\/$/, '');
    const response = await fetch(`${base}/api/v1/auth/${action}`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: await request.text(), cache: 'no-store', signal: AbortSignal.timeout(15_000),
    });
    const data = await response.json().catch(() => ({ error: {
      message: response.ok ? 'The server returned an invalid response.' :
        action === 'login' ? 'Unable to log in. Please check your email and password.' : 'Unable to create your account. Please try again.',
    } }));
    return NextResponse.json(data, {
      status: response.ok && data.error ? 502 : response.status,
      headers: { 'Cache-Control': 'no-store' },
    });
  } catch {
    return NextResponse.json({ error: { message: 'We could not reach Finos. Please try again in a moment.' } },
      { status: 503, headers: { 'Cache-Control': 'no-store' } });
  }
}
