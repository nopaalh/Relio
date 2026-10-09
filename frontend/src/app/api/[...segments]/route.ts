import { NextRequest, NextResponse } from "next/server";
import { context, listDeals, evidence } from "@/lib/server/local-data";
import { assertDate, validateSelection } from "@/lib/domain";
import type { Comparison } from "@/lib/contracts";
export const runtime="nodejs";
export const dynamic="force-dynamic";
const reply=(body:unknown,status=200)=>NextResponse.json(body,{status,headers:{"Cache-Control":"no-store"}});
async function proxy(req:NextRequest,segments:string[]) {
  const base=process.env.RELIO_API_URL;
  if(!base)return reply({error:"RELIO_API_URL is not configured."},503);
  const url=new URL("/api/"+segments.map(encodeURIComponent).join("/"),base);
  url.search=req.nextUrl.search;
  try{
    const res=await fetch(url,{method:req.method,headers:{"Content-Type":"application/json"},
      ...(req.method==="POST"?{body:await req.text()}:{}),cache:"no-store",signal:AbortSignal.timeout(45000)});
    return new NextResponse(await res.text(),{status:res.status,headers:{"Content-Type":"application/json","Cache-Control":"no-store"}});
  }catch{return reply({error:"Backend unavailable. Check RELIO_API_URL."},503);}
}
type RouteContext={params:Promise<{segments:string[]}>};
export async function GET(req:NextRequest,{params}:RouteContext) {
  const {segments:s}=await params;
  if(process.env.RELIO_DATA_MODE==="api")return proxy(req,s);
  const date=req.nextUrl.searchParams.get("as_of")??"2026-10-01";
  try{
    assertDate(date);
    if(s[0]==="health")return reply({mode:"local",data:"available",jev:"unavailable",copilot:"unavailable",snapshot_date:"2026-10-01"});
    if(s[0]==="evidence"&&s.length===2)return reply(await evidence(s[1],date));
    if(s[0]==="deals"&&s.length===1)return reply(await listDeals(date));
    if(s[0]==="deals"&&s[1]){
      const data=await context(s[1],date);
      if(s.length===2)return reply({scope:data.scope,deal:data.deal});
      if(s[2]==="timeline")return reply({events:data.timeline,navigation:data.navigation});
      if(s[2]==="graph")return reply(data.graph);
      if(s[2]==="assessment")return reply(data.assessment);
      if(s[2]==="action-candidates")return reply({candidates:data.candidates,catalogue_version:data.scope.evidence_version});
    }
    return reply({error:"Endpoint not found."},404);
  }catch(e){const message=e instanceof Error?e.message:"Local data unavailable.";return reply({error:message},/not found/i.test(message)?404:/date/i.test(message)?400:503);}
}
export async function POST(req:NextRequest,{params}:RouteContext) {
  const {segments:s}=await params;
  if(process.env.RELIO_DATA_MODE==="api")return proxy(req,s);
  try{
    const body=await req.json();const date=body.as_of??"2026-10-01";assertDate(date);
    if(s[0]==="copilot"&&s[1]==="ask") {
      if(typeof body.question!=="string"||!body.question.trim()||body.question.length>2000)return reply({error:"Enter a question of 1–2000 characters."},400);
      return reply({answer:"Copilot is not connected to retrieval and a model yet. You can still explore the timeline, planets, and source evidence. No AI answer is available for this question.",
        status:"unavailable",evidence_ids:[],unknowns:["The backend Copilot service is unavailable."],as_of:date,deal_id:body.deal_id??null});
    }
    if(s[0]==="deals"&&s[2]==="actions"&&s[3]==="compare"){
      const data=await context(s[1],date);
      if(!Array.isArray(body.action_ids)||!body.action_ids.every((id:unknown)=>typeof id==="string"))return reply({error:"action_ids must be an array of IDs."},400);
      const error=validateSelection(body.action_ids,data.candidates.map(c=>c.action_id));if(error)return reply({error},400);
      const result:Comparison={deal_id:s[1],as_of:date,comparison_type:"action_suitability_not_win_probability",assessment_status:"jev_unavailable",rubric_version:"action-suitability-v1.0",
        items:body.action_ids.map((id:string)=>({action_id:id,suitability_score_100:null,jev_raw_score:null,jev_confidence:null,jev_choice_preference:null,rank:null,
          evidence_ids:data.candidates.find(c=>c.action_id===id)!.occurrences.flatMap(o=>o.evidence_ids),precedent_ids:data.candidates.find(c=>c.action_id===id)!.occurrences.map(o=>o.occurrence_id),
          policy_flags:data.candidates.find(c=>c.action_id===id)!.policy_flags,unknowns:["JEV is not connected. No ranking can be inferred."]})),
        warning:"Suitability is not Closed Won probability. Choice is a relative preference within the selected set."};
      return reply(result);
    }
    return reply({error:"Endpoint not found."},404);
  }catch(e){return reply({error:e instanceof Error?e.message:"Invalid request."},400);}
}
