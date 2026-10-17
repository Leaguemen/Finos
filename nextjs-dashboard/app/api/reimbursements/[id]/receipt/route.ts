import { NextRequest, NextResponse } from 'next/server';

export async function GET(request: NextRequest, context: { params: Promise<{ id: string }> }) {
  const headers = { 'Cache-Control': 'no-store' };
  const authorization = request.headers.get('authorization');
  if (!authorization?.startsWith('Bearer ')) return NextResponse.json({ error: { message: 'Authentication is required' } }, { status: 401, headers });
  const { id } = await context.params;
  if (!/^[1-9]\d*$/.test(id)) return NextResponse.json({ error: { message: 'Reimbursement ID must be a positive integer' } }, { status: 400, headers });
  try {
    const base = (process.env.FINOS_API_URL || 'http://localhost:8080').replace(/\/$/, '');
    const authorized = await fetch(`${base}/api/v1/reimbursements/${id}/receipt`, {
      headers: { Authorization: authorization }, redirect: 'manual', cache: 'no-store', signal: AbortSignal.timeout(15_000),
    });
    if (authorized.status >= 300 && authorized.status < 400) {
      const location = authorized.headers.get('location');
      if (!location) return NextResponse.json({ error: { message: 'The receipt link was missing.' } }, { status: 502, headers });
      // Fetch the signed storage URL without the Finos bearer token.
      const receipt = await fetch(location, { cache: 'no-store', signal: AbortSignal.timeout(20_000) });
      if (!receipt.ok) return NextResponse.json({ error: { message: 'The receipt could not be downloaded.' } }, { status: 502, headers });
      return new NextResponse(await receipt.arrayBuffer(), {
        status: 200,
        headers: { ...headers, 'Content-Type': receipt.headers.get('content-type') || 'application/octet-stream', 'Content-Disposition': 'inline' },
      });
    }
    const payload = await authorized.json().catch(() => null);
    return NextResponse.json(payload ?? { error: { message: 'The server returned an invalid response.' } }, { status: payload ? authorized.status : 502, headers });
  } catch {
    return NextResponse.json({ error: { message: 'We could not open this receipt. Please try again.' } }, { status: 503, headers });
  }
}
