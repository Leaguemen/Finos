import { NextRequest, NextResponse } from 'next/server';
async function forward(request: NextRequest) {
  const authorization = request.headers.get('authorization');
  const headers = { 'Cache-Control': 'no-store' };
  if (!authorization?.startsWith('Bearer ')) return NextResponse.json({ error: { message: 'Authentication is required' } }, { status: 401, headers });
  try {
    const base = (process.env.FINOS_API_URL || 'http://localhost:8080').replace(/\/$/, '');
    const upstream = await fetch(`${base}/api/v1/timesheets`, {
      method: request.method, headers: { Authorization: authorization, 'Content-Type': 'application/json' },
      ...(request.method === 'POST' ? { body: await request.text() } : {}),
      cache: 'no-store', signal: AbortSignal.timeout(15_000),
    });
    const body = await upstream.json().catch(() => null);
    return NextResponse.json(body ?? { error: { message: 'The server returned an invalid response.' } }, { status: body ? upstream.status : 502, headers });
  } catch {
    return NextResponse.json({ error: { message: 'We could not reach Finos. Please try again in a moment.' } }, { status: 503, headers });
  }
}
export const GET = forward;
export const POST = forward;
