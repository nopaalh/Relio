import test from "node:test";
import assert from "node:assert/strict";
import { DEFAULT_API_URL, PROXY_TIMEOUT_MS, proxyRequest } from "../src/lib/server/proxy.ts";

// DATA UJI: fetch is injected; no backend, database, provider or network is called.
test("DATA UJI: default Go proxy forwards encoded path/query and truthful status/content type", async () => {
  let calls = 0;
  const fetcher: typeof fetch = async (input, init) => {
    calls++;
    assert.equal(String(input), DEFAULT_API_URL + "/api/deals/DATA%20UJI%2Fid/timeline?as_of=2026-09-01&limit=50&cursor=DATA%2BUJI&as_of=2026-09-01");
    assert.equal(init?.method, "GET"); assert.equal(init?.cache, "no-store"); assert.equal(init?.redirect, "manual");
    assert.equal(init?.credentials, "omit"); assert.ok(init?.signal); assert.equal(PROXY_TIMEOUT_MS, 130_000);
    assert.equal(new Headers(init?.headers).get("Authorization"), null); assert.equal(new Headers(init?.headers).get("Cookie"), null);
    return new Response("DATA UJI plain upstream failure", { status: 404, headers: { "Content-Type": "text/plain; charset=utf-8", "Set-Cookie": "DATA-UJI-only" } });
  };
  const req = new Request("http://frontend.invalid/api/data-uji?as_of=2026-09-01&limit=50&cursor=DATA%2BUJI&as_of=2026-09-01", {
    headers: { Authorization: "DATA-UJI-not-a-secret", Cookie: "DATA-UJI-only" },
  });
  const result = await proxyRequest(req, ["deals", "DATA UJI/id", "timeline"], undefined, fetcher);
  assert.equal(calls, 1); assert.equal(result.status, 404); assert.match(result.headers.get("Content-Type")!, /^text\/plain/);
  assert.equal(result.headers.get("Set-Cookie"), null); assert.equal(result.headers.get("Cache-Control"), "no-store");
  assert.equal(await result.text(), "DATA UJI plain upstream failure");
});

test("DATA UJI: POST body and content type forward unchanged, without user auth headers", async () => {
  const body = '{"as_of":"2026-09-01","action_ids":["DATA-UJI:a","DATA-UJI:b"]}';
  const fetcher: typeof fetch = async (input, init) => {
    assert.equal(String(input), "https://backend.invalid/api/deals/DATA-UJI/actions/compare");
    assert.equal(init?.method, "POST"); assert.equal(init?.body, body);
    const headers = new Headers(init?.headers); assert.equal(headers.get("Content-Type"), "application/json"); assert.equal(headers.get("X-Api-Key"), null);
    return Response.json({ code: "DATA-UJI-unavailable", message: "DATA UJI provider unavailable" }, { status: 503 });
  };
  const result = await proxyRequest(new Request("http://frontend.invalid/api/deals/DATA-UJI/actions/compare", {
    method: "POST", headers: { "Content-Type": "application/json", "X-Api-Key": "DATA-UJI-not-a-secret" }, body,
  }), ["deals", "DATA-UJI", "actions", "compare"], "https://backend.invalid", fetcher);
  assert.equal(result.status, 503); assert.match(result.headers.get("Content-Type")!, /application\/json/);
  assert.equal((await result.json()).code, "DATA-UJI-unavailable");
});

test("DATA UJI: invalid/credential-bearing URL gives friendly 503 without URL or exception details", async () => {
  for (const url of ["not a URL", "ftp://backend.invalid", "http://DATA-UJI:DATA-UJI@backend.invalid", "https://backend.invalid?DATA-UJI=query", "https://backend.invalid/api"]) {
    let called = false;
    const fetcher: typeof fetch = async () => { called = true; throw new Error("DATA UJI should not execute"); };
    const result = await proxyRequest(new Request("http://frontend.invalid/api/health"), ["health"], url, fetcher);
    assert.equal(result.status, 503); assert.equal(called, false);
    const error = await result.json(); assert.equal(error.code, "invalid_backend_url"); assert.ok(!error.message.includes(url));
  }
});

test("DATA UJI: redirects are blocked and Location/auth headers never reach browser", async () => {
  let calls = 0;
  const fetcher: typeof fetch = async (_input, init) => {
    calls++; assert.equal(init?.redirect, "manual");
    return new Response(null, { status: 302, headers: { Location: "https://other.invalid/DATA-UJI", "Set-Cookie": "DATA-UJI-only" } });
  };
  const result = await proxyRequest(new Request("http://frontend.invalid/api/health"), ["health"], undefined, fetcher);
  assert.equal(calls, 1); assert.equal(result.status, 502); assert.equal(result.headers.get("Location"), null);
  assert.equal(result.headers.get("Set-Cookie"), null); assert.equal((await result.json()).code, "backend_redirect");
});

test("DATA UJI: health alias routes to /api/health and 204 remains bodyless", async () => {
  const fetcher: typeof fetch = async input => {
    assert.equal(String(input), DEFAULT_API_URL + "/api/health");
    return new Response(null, { status: 204 });
  };
  const result = await proxyRequest(new Request("http://frontend.invalid/api/health"), ["health"], undefined, fetcher);
  assert.equal(result.status, 204); assert.equal(await result.text(), ""); assert.equal(result.headers.get("Content-Type"), null);
});

test("DATA UJI: the bounded fetch timeout produces a safe 504 without waiting or networking", async t => {
  t.mock.method(AbortSignal, "timeout", (milliseconds: number) => {
    assert.equal(milliseconds, PROXY_TIMEOUT_MS);
    return AbortSignal.abort(new DOMException("DATA UJI timeout details", "TimeoutError"));
  });
  const fetcher: typeof fetch = async (_input, init) => {
    assert.equal(init?.signal?.aborted, true); throw init?.signal?.reason;
  };
  const result = await proxyRequest(new Request("http://frontend.invalid/api/health"), ["health"], undefined, fetcher);
  assert.equal(result.status, 504);
  const body = await result.json(); assert.equal(body.code, "backend_timeout"); assert.ok(!body.message.includes("DATA UJI"));
});

test("DATA UJI: network errors are safe JSON, with one attempt and no internal details", async () => {
  let calls = 0;
  const fetcher: typeof fetch = async () => { calls++; throw new Error("DATA UJI internal socket/configuration details"); };
  const result = await proxyRequest(new Request("http://frontend.invalid/api/health"), ["health"], undefined, fetcher);
  assert.equal(result.status, 503); assert.equal(calls, 1);
  const error = await result.json(); assert.equal(error.code, "backend_unavailable"); assert.ok(!error.message.includes("socket"));
});
