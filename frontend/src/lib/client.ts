import type { ContextBundle, DealsResponse, Evidence, Comparison, CopilotResponse } from "./contracts";
export async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(url,{...options,cache:"no-store",headers:{"Content-Type":"application/json",...options?.headers}});
  const data = await res.json();
  if (!res.ok) throw new Error(data.error ?? "Service unavailable. Please try again.");
  return data as T;
}
export const api = {
  deals: (date:string, signal?:AbortSignal) => request<DealsResponse>("/api/deals?as_of="+date,{signal}),
  context: async (id:string,date:string,signal?:AbortSignal):Promise<ContextBundle> => {
    const base="/api/deals/"+encodeURIComponent(id), suffix="?as_of="+date;
    const [metadata,timeline,graph,assessment,catalogue]=await Promise.all([
      request<{scope:ContextBundle["scope"];deal:ContextBundle["deal"]}>(base+suffix,{signal}),
      request<{events:ContextBundle["timeline"];navigation?:ContextBundle["navigation"]}>(base+"/timeline"+suffix,{signal}),
      request<ContextBundle["graph"]>(base+"/graph"+suffix+"&depth=1",{signal}),
      request<ContextBundle["assessment"]>(base+"/assessment"+suffix,{signal}),
      request<{candidates:ContextBundle["candidates"];catalogue_version?:string}>(base+"/action-candidates"+suffix,{signal})
    ]);
    return {catalogue_version:catalogue.catalogue_version,navigation:timeline.navigation,scope:metadata.scope,deal:metadata.deal,timeline:timeline.events,graph,assessment,candidates:catalogue.candidates};
  },
  evidence: (id:string,date:string,signal?:AbortSignal) => request<Evidence>("/api/evidence/"+encodeURIComponent(id)+"?as_of="+date,{signal}),
  compare: (id:string,date:string,ids:string[],signal?:AbortSignal) => request<Comparison>("/api/deals/"+encodeURIComponent(id)+"/actions/compare",{method:"POST",body:JSON.stringify({as_of:date,action_ids:ids}),signal}),
  ask: (question:string,date:string,id:string|null,signal?:AbortSignal) => request<CopilotResponse>("/api/copilot/ask",{method:"POST",body:JSON.stringify({question,as_of:date,deal_id:id}),signal})
};
