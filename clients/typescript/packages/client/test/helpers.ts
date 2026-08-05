export type MockReply = {
  status?: number;
  body?: unknown;
  headers?: Record<string, string>;
};

export type Recorded = {
  url: string;
  method: string;
  body: unknown;
  headers: Headers;
};

export function mockFetch(reply: (req: Recorded) => MockReply) {
  const calls: Recorded[] = [];
  const impl = (async (input: RequestInfo | URL, init?: RequestInit) => {
    const rawBody = init?.body;
    const rec: Recorded = {
      url: String(input),
      method: init?.method ?? "GET",
      body: typeof rawBody === "string" && rawBody.length > 0 ? JSON.parse(rawBody) : undefined,
      headers: new Headers(init?.headers),
    };
    calls.push(rec);
    const out = reply(rec);
    const payload = out.status === 204 ? null : out.body === undefined ? "" : JSON.stringify(out.body);
    const responseInit: ResponseInit = { status: out.status ?? 200 };
    if (out.headers) {
      responseInit.headers = out.headers;
    }
    return new Response(payload, responseInit);
  }) as typeof fetch;
  return { impl, calls };
}

export const settle = (): Promise<void> => new Promise((resolve) => setTimeout(resolve, 0));
