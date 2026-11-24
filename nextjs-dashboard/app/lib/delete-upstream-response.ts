import { NextResponse } from 'next/server';

const headers = { 'Cache-Control': 'no-store' };

export async function deleteUpstreamResponse(upstream: Response, resource: string) {
  if (upstream.status === 204) return new NextResponse(null, { status: 204, headers });

  const body = await upstream.json().catch(() => null);
  if (!upstream.ok && body && typeof body?.error?.message === 'string') {
    return NextResponse.json(body, { status: upstream.status, headers });
  }

  console.error('Unexpected Finos API DELETE response', {
    resource,
    status: upstream.status,
    contentType: upstream.headers.get('content-type'),
  });
  if (upstream.status === 404 || upstream.status === 405) {
    return NextResponse.json({ error: {
      code: 'upstream_route_unavailable',
      message: 'The backend delete endpoint is unavailable. Deploy the latest Go API and check FINOS_API_URL.',
    } }, { status: 502, headers });
  }
  return NextResponse.json({ error: {
    code: 'upstream_invalid_response',
    message: `The backend returned an unexpected response (HTTP ${upstream.status}). Please try again or check the Go API logs.`,
  } }, { status: 502, headers });
}
