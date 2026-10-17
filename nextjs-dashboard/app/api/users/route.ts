import { NextRequest, NextResponse } from 'next/server';

export async function GET(request: NextRequest) {
  const headers = { 'Cache-Control': 'no-store' };
  const authorization = request.headers.get('authorization');

  if (!authorization?.startsWith('Bearer ')) {
    return NextResponse.json(
      { error: { message: 'Authentication is required' } },
      { status: 401, headers },
    );
  }

  try {
    const base = (process.env.FINOS_API_URL || 'http://localhost:8080').replace(/\/$/, '');
    const upstream = await fetch(`${base}/api/v1/users`, {
      headers: { Authorization: authorization },
      cache: 'no-store',
      signal: AbortSignal.timeout(15_000),
    });
    const body = await upstream.json().catch(() => null);

    return NextResponse.json(
      body ?? { error: { message: 'The server returned an invalid response.' } },
      { status: body ? upstream.status : 502, headers },
    );
  } catch {
    return NextResponse.json(
      { error: { message: 'We could not reach Finos. Please try again in a moment.' } },
      { status: 503, headers },
    );
  }
}
