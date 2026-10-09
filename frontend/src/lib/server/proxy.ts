export const DEFAULT_API_URL = "http://127.0.0.1:8080";
export const PROXY_TIMEOUT_MS = 130_000;

const error = (status: number, code: string, message: string) => Response.json({ code, message }, {
  status, headers: { "Cache-Control": "no-store" },
});

// Only transport headers cross this boundary, never browser auth or backend cookies.
export async function proxyRequest(request: Request, segments: string[], backendUrl = DEFAULT_API_URL, fetcher: typeof fetch = fetch): Promise<Response> {
  let url: URL;
  try {
    const base = new URL(backendUrl);
    if (!["http:", "https:"].includes(base.protocol) || base.username || base.password || base.search || base.hash || base.pathname !== "/") throw new Error();
    if (!segments.length || segments.some(s => !s.trim() || s === "." || s === "..")) return error(400, "invalid_path", "Invalid API path.");
    url = new URL("/api/" + segments.map(encodeURIComponent).join("/"), base);
    url.search = new URL(request.url).search;
  } catch {
    return error(503, "invalid_backend_url", "Backend URL is not configured correctly. Use an HTTP(S) origin without credentials.");
  }
  const timeout = AbortSignal.timeout(PROXY_TIMEOUT_MS);
  const signal = AbortSignal.any([request.signal, timeout]);
  try {
    const headers = new Headers();
    const contentType = request.headers.get("Content-Type");
    if (contentType) headers.set("Content-Type", contentType);
    const response = await fetcher(url, {
      method: request.method, headers, cache: "no-store", credentials: "omit", redirect: "manual", signal,
      ...(request.method === "POST" ? { body: await request.text() } : {}),
    });
    if (response.redirected || (response.status >= 300 && response.status < 400)) {
      return error(502, "backend_redirect", "Backend redirects are not supported. Check the backend origin.");
    }
    const replyHeaders = new Headers({ "Cache-Control": "no-store" });
    const responseType = response.headers.get("Content-Type");
    if (responseType) replyHeaders.set("Content-Type", responseType);
    const body = response.status === 204 || response.status === 205 ? null : await response.arrayBuffer();
    return new Response(body, { status: response.status, headers: replyHeaders });
  } catch {
    if (timeout.aborted) return error(504, "backend_timeout", "Backend request timed out. Please try again.");
    return error(503, "backend_unavailable", "Backend unavailable. Check the server configuration and try again.");
  }
}
