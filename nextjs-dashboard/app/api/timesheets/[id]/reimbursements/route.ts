import { NextRequest, NextResponse } from 'next/server';

async function forward(request: NextRequest, context: { params: Promise<{ id: string }> }) {
  const responseHeaders = { 'Cache-Control': 'no-store' };
  const authorization = request.headers.get('authorization');
  if (!authorization?.startsWith('Bearer ')) return NextResponse.json({ error: { message: 'Authentication is required' } }, { status: 401, headers: responseHeaders });
  const { id } = await context.params;
  if (!/^[1-9]\d*$/.test(id)) return NextResponse.json({ error: { message: 'Timesheet ID must be a positive integer' } }, { status: 400, headers: responseHeaders });
  try {
    const base = (process.env.FINOS_API_URL || 'http://localhost:8080').replace(/\/$/, '');
    const headers: HeadersInit = { Authorization: authorization };
    let body: ArrayBuffer | undefined;
    if (request.method === 'POST') {
      const contentType = request.headers.get('content-type');
      if (contentType) headers['Content-Type'] = contentType;
      body = await request.arrayBuffer();
    }
    const upstream = await fetch(`${base}/api/v1/timesheets/${id}/reimbursements`, {
      method: request.method, headers, body, cache: 'no-store', signal: AbortSignal.timeout(30_000),
    });
    const payload = await upstream.json().catch(() => null);
    return NextResponse.json(payload ?? { error: { message: 'The server returned an invalid response.' } }, { status: payload ? upstream.status : 502, headers: responseHeaders });
  } catch {
    return NextResponse.json({ error: { message: 'We could not reach Finos. Please try again in a moment.' } }, { status: 503, headers: responseHeaders });
  }
}

export const GET = forward;
export const POST = forward;
