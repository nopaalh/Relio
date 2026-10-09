import type {
  ActionCandidate, Assessment, Comparison, ContextBundle, CopilotResponse, DealPage, DealResult,
  DealsResponse, Evidence, EvidenceResult, FeatureState, GraphResult, SnapshotMeta, TimelineResult, TimelineView,
} from "./contracts.ts";
import { applyAssessment, assertDealResult, assertMeta, consumePages, toDeal, toEvent, toEvidence, toGraph, toScope } from "./adapters.ts";

export class ApiError extends Error {
  status: number;
  code: string;
  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

export async function request<T>(url: string, options?: RequestInit): Promise<T> {
  let res: Response;
  try {
    const headers = new Headers(options?.headers);
    if (!headers.has("Content-Type")) headers.set("Content-Type", "application/json");
    res = await fetch(url, { ...options, headers, cache: "no-store" });
  } catch (e) {
    if (options?.signal?.aborted) throw e;
    throw new ApiError(0, "network_error", "Backend request could not be completed. Please try again.");
  }
  let data: unknown;
  const json = /\bapplication\/(?:[\w.-]+\+)?json\b/i.test(res.headers.get("Content-Type") ?? "");
  try { if (json) data = await res.json(); } catch { /* Invalid bodies get a safe transport error, not their raw text. */ }
  if (!res.ok) {
    const body = data && typeof data === "object" ? data as Record<string, unknown> : {};
    const message = typeof body.message === "string" ? body.message : typeof body.error === "string" ? body.error : `Service unavailable (HTTP ${res.status}). Please try again.`;
    throw new ApiError(res.status, typeof body.code === "string" ? body.code : "http_error", message.slice(0, 500));
  }
  if (data === undefined || data === null) throw new ApiError(res.status, "invalid_response", "Backend returned an invalid or non-JSON response.");
  return data as T;
}

export type OptionalRead<T> = { data: T | null; state: FeatureState };
export async function optionalRead<T extends { meta?: SnapshotMeta }>(url: string, signal?: AbortSignal, expected?: SnapshotMeta, validate?: (data: T) => void): Promise<OptionalRead<T>> {
  try {
    const data = await request<T>(url, { signal });
    if (expected) assertMeta(data.meta!, expected.as_of, expected);
    validate?.(data);
    return { data, state: { status: "available" } };
  } catch (e) {
    if (signal?.aborted) throw e;
    return { data: null, state: {
      status: e instanceof ApiError && e.status === 503 ? "unavailable" : "error",
      message: e instanceof Error ? e.message : "Feature request failed.",
      ...(e instanceof ApiError ? { code: e.code, http_status: e.status } : {}),
    } };
  }
}

const suffix = (date: string) => "?as_of=" + encodeURIComponent(date);
function assertLocal(scope: DealsResponse["scope"], date: string) {
  if (!scope || scope.mode !== "local" || scope.as_of !== date) throw new Error("Unexpected backend envelope or snapshot mismatch.");
}

async function timeline(id: string, date: string, signal?: AbortSignal, expected?: SnapshotMeta): Promise<TimelineView> {
  const url = "/api/deals/" + encodeURIComponent(id) + "/timeline" + suffix(date) + "&limit=50";
  const first = await request<TimelineResult | TimelineView>(url, { signal });
  if ("meta" in first) {
    const page = first as TimelineResult;
    const result = await consumePages(async cursor => {
      const data = cursor === undefined ? page : await request<TimelineResult>(url + "&cursor=" + encodeURIComponent(cursor), { signal });
      return { meta: data.meta, bounds: data.bounds, values: data.events };
    }, date, expected);
    const events = result.values.map(e => toEvent(e, date));
    if (new Set(events.map(e => e.event_id)).size !== events.length) throw new Error("Backend repeated timeline events across pages.");
    events.sort((a, b) => a.date.localeCompare(b.date) || a.event_id.localeCompare(b.event_id));
    return { events, meta: result.meta, bounds: result.bounds, pages: result.pages };
  }
  if (expected || !Array.isArray(first.events)) throw new Error("Expected a canonical timeline envelope.");
  return first as TimelineView;
}

export const api = {
  deals: async (date: string, signal?: AbortSignal): Promise<DealsResponse> => {
    const url = "/api/deals" + suffix(date) + "&limit=50";
    const first = await request<DealPage | DealsResponse>(url, { signal });
    if ("meta" in first) {
      const result = await consumePages(async cursor => {
        const page = cursor === undefined ? first : await request<DealPage>(url + "&cursor=" + encodeURIComponent(cursor), { signal });
        return { meta: page.meta, bounds: page.bounds, values: page.items };
      }, date);
      const deals = result.values.map(f => toDeal(f, result.meta));
      if (new Set(deals.map(d => d.deal_id)).size !== deals.length) throw new Error("Backend repeated deals across pages.");
      return { scope: toScope(result.meta, result.pages.slice(1).map(p => p.meta)), deals, bounds: result.bounds, pages: result.pages };
    }
    assertLocal(first.scope, date);
    if (!Array.isArray(first.deals)) throw new Error("Invalid local deal list.");
    return first;
  },
  timeline,
  context: async (id: string, date: string, signal?: AbortSignal): Promise<ContextBundle> => {
    const base = "/api/deals/" + encodeURIComponent(id), query = suffix(date);
    const metadata = await request<DealResult | { scope: ContextBundle["scope"]; deal: ContextBundle["deal"] }>(base + query, { signal });
    const canonical = "meta" in metadata;
    if (canonical) assertDealResult(metadata, id, date);
    else assertLocal(metadata.scope, date);
    const expected = canonical ? metadata.meta : undefined;
    const deal = canonical ? toDeal(metadata.deal, metadata.meta) : metadata.deal;
    const [events, graphResponse, assessment, catalogue] = await Promise.all([
      timeline(id, date, signal, expected),
      request<GraphResult | ContextBundle["graph"]>(base + "/graph" + query + "&depth=2&max_nodes=150&max_edges=300", { signal }),
      optionalRead<Assessment>(base + "/assessment" + query, signal, expected, data => {
        if (typeof data.status !== "string" || typeof data.explanation !== "string" || typeof data.rubric_version !== "string" || !Array.isArray(data.unknowns)) throw new Error("Invalid assessment response.");
        applyAssessment(deal, data);
      }),
      optionalRead<{ meta?: SnapshotMeta; candidates: ActionCandidate[]; catalogue_version?: string }>(base + "/action-candidates" + query, signal, expected, data => {
        if (!Array.isArray(data.candidates)) throw new Error("Unsupported action catalogue response. No valid zero-result catalogue can be inferred.");
      }),
    ]);
    let graph: ContextBundle["graph"];
    if (canonical) {
      const result = graphResponse as GraphResult;
      assertMeta(result.meta, date, expected);
      graph = toGraph(result);
    } else {
      if ("meta" in graphResponse || !Array.isArray(graphResponse.nodes) || !Array.isArray(graphResponse.edges)) throw new Error("Unexpected local graph response.");
      graph = graphResponse as ContextBundle["graph"];
    }
    const otherMeta = [...(events.pages?.map(p => p.meta) ?? []), ...(graph.meta ? [graph.meta] : []), ...(assessment.data?.meta ? [assessment.data.meta] : []), ...(catalogue.data?.meta ? [catalogue.data.meta] : [])];
    return {
      scope: canonical ? toScope(metadata.meta, otherMeta) : metadata.scope,
      deal: canonical ? applyAssessment(deal, assessment.data) : deal, timeline: events.events,
      timeline_meta: events.meta, timeline_bounds: events.bounds, timeline_pages: events.pages, navigation: events.navigation,
      graph, assessment: assessment.data, assessment_state: assessment.state,
      candidates: catalogue.data?.candidates ?? null, candidates_state: catalogue.state, catalogue_version: catalogue.data?.catalogue_version,
    };
  },
  evidence: async (id: string, date: string, signal?: AbortSignal, expected?: SnapshotMeta): Promise<Evidence> => {
    const result = await request<EvidenceResult | Evidence>("/api/evidence/" + encodeURIComponent(id) + suffix(date), { signal });
    if ("meta" in result && "evidence" in result) {
      const record = toEvidence(result, date, expected);
      if (record.evidence_id !== id) throw new Error("Backend returned a different evidence unit than requested.");
      return record;
    }
    if (expected || !("raw_record" in result) || result.evidence_id !== id) throw new Error("Expected scoped backend evidence.");
    return result;
  },
  compare: (id: string, date: string, ids: string[], signal?: AbortSignal) => request<Comparison>("/api/deals/" + encodeURIComponent(id) + "/actions/compare", { method: "POST", body: JSON.stringify({ as_of: date, action_ids: ids }), signal }),
  ask: (question: string, date: string, id: string | null, signal?: AbortSignal) => request<CopilotResponse>("/api/copilot/ask", { method: "POST", body: JSON.stringify({ question, as_of: date, deal_id: id }), signal }),
};
