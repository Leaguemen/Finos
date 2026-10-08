import { NextRequest, NextResponse } from 'next/server';

export async function PATCH(request: NextRequest, context: { params: Promise<{ id: string }> }) {
  const headers = { 'Cache-Control': 'no-store' };
  const authorization = request.headers.get('authorization');
  if (!authorization?.startsWith('Bearer ')) return NextResponse.json({ error: { message: 'Authentication is required' } }, { status: 401, headers });
  const { id } = await context.params;
  if (!/^[1-9]\d*$/.test(id)) return NextResponse.json({ error: { message: 'Timesheet ID must be a positive integer' } }, { status: 400, headers });
  try {
    const base = (process.env.FINOS_API_URL || 'http://localhost:8080').replace(/\/$/, '');
    const upstream = await fetch(`${base}/api/v1/timesheets/${id}/approve`, {
      method: 'PATCH', headers: { Authorization: authorization },
      cache: 'no-store', signal: AbortSignal.timeout(15_000),
    });
    const body = await upstream.json().catch(() => null);
    return NextResponse.json(body ?? { error: { message: 'The server returned an invalid response.' } }, { status: body ? upstream.status : 502, headers });
  } catch {
    return NextResponse.json({ error: { message: 'We could not reach Finos. Please try again in a moment.' } }, { status: 503, headers });
  }
}
