import fs from "node:fs/promises";
import path from "node:path";
import { createHash } from "node:crypto";
import { parse } from "csv-parse/sync";
import { display } from "../display";
import { assertDate, attractiveness, visibleAt } from "../domain";
import type { ActionCandidate, ContextBundle, Deal, Evidence, TimelineEvent, GraphNode, GraphEdge, Scope } from "../contracts";
type Row = Record<string,string>;
const dataDir=path.resolve(process.cwd(),process.env.DATA_DIR??"../dataset_kasirnusa");
const snapshot="2026-10-01";
let cached: Promise<{accounts:Row[];deals:Row[];employees:Row[];contacts:Row[];history:Row[];decisions:Row[];interactions:Row[];version:string}>|null=null;
async function csv(file:string):Promise<Row[]> {return parse(await fs.readFile(path.join(dataDir,file),"utf8"),{columns:true,bom:true,skip_empty_lines:true});}
export function dataset() {
  return cached??= (async()=>{
    const [accounts,deals,employees,contacts,history,decisions,raw]=await Promise.all([
      csv("crm_accounts.csv"),csv("crm_deals.csv"),csv("employees.csv"),csv("crm_contacts.csv"),
      csv("contact_employment_history.csv"),csv("decision_log.csv"),fs.readFile(path.join(dataDir,"interactions.jsonl"),"utf8")
    ]);
    const interactions:Row[]=raw.trim().split(/\r?\n/).map(l=>JSON.parse(l));
    const version=createHash("sha256").update(JSON.stringify({accounts,deals,employees,contacts,history,decisions,interactions})).digest("hex").slice(0,16);
    return {accounts,deals,employees,contacts,history,decisions,interactions,version};
  })();
}
function scope(date:string,version:string):Scope{return {as_of:date,snapshot_date:snapshot,evidence_version:version,mode:"local",
  limitations:["Synthetic KasirNusa dataset; read-only source projection.","JEV and Copilot are not connected. Historical options need relevance validation.","CRM snapshot facts are not backdated."]};}
const eventTypes:Record<string,string>={I0269:"MEETING",I0279:"MEETING",I0284:"MEETING",I0296:"MEETING",I0310:"MEETING",I0314:"MEETING",I0322:"FOLLOW_UP",I0325:"PROPOSAL",I0334:"BUYER_REQUEST",I0335:"BUYER_REQUEST",I0343:"INBOUND_EMAIL",I0348:"DISCOUNT_REQUEST"};
function actorOf(row:Row,data:Awaited<ReturnType<typeof dataset>>):string {
  const employee=data.employees.find(e=>e.email===row.dari);
  if(employee)return employee.nama;
  // Match the address used in this source. Do not infer employment from today's CRM account.
  const contact=data.contacts.find(c=>c.email===row.dari);
  return contact?.nama??row.dari??"Unknown actor";
}
function events(account:string,date:string,data:Awaited<ReturnType<typeof dataset>>):TimelineEvent[] {
  return data.interactions.filter(i=>i.account_id===account&&visibleAt(i.tanggal,date)).map(i=>({
    event_id:"EV-"+i.interaction_id,interaction_id:i.interaction_id,date:i.tanggal,title:i.subjek,
    type:eventTypes[i.interaction_id]??(i.tipe==="catatan_meeting"?"MEETING":"OTHER"),
    actor:actorOf(i,data),actor_email:i.dari,status:i.interaction_id==="I0348"?"requested":"unknown",
    excerpt:i.isi,evidence_ids:["EVI-"+i.interaction_id],verification:"verified" as const
  })).sort((a,b)=>a.date.localeCompare(b.date)||a.interaction_id.localeCompare(b.interaction_id));
}
const templates=[
  {id:"HIST-PILOT-STARTER",title:"Starter pilot",decision:"D-2025-06",description:"A six-outlet pilot with no discount, sourced from the decision in case C23."},
  {id:"HIST-PREPAID-TERM",title:"Contract commitment discount",decision:"D-2024-01",description:"A 12% discount precedent for a prepaid two-year contract."},
  {id:"HIST-PUBLIC-REFERENCE",title:"Public reference agreement",decision:"D-2025-09",description:"A 12% discount precedent with a public reference in the F&B industry."},
  {id:"HIST-TRAINING",title:"Additional training",decision:"D-2026-07",description:"A precedent for free additional training when cashiers change."}
];
function candidates(date:string,data:Awaited<ReturnType<typeof dataset>>):ActionCandidate[] {
  return templates.flatMap(t=>{
    const d=data.decisions.find(d=>d.decision_id===t.decision&&visibleAt(d.tanggal,date));
    if(!d)return [];
    const discount=Number.parseFloat(d.nilai);
    return [{action_id:t.id,title:t.title,description:t.description,eligibility:"requires_validation" as const,
      occurrences:[{occurrence_id:"AO-"+d.decision_id,account_id:d.account_id,account_name:data.accounts.find(a=>a.account_id===d.account_id)?.nama??d.account_id,
        date:d.tanggal,status:d.keputusan==="Disetujui"?"approved":d.keputusan==="Ditolak"?"rejected":"requested",outcome_observed:null,evidence_ids:["EVD-"+d.decision_id]}],
      policy_flags:Number.isFinite(discount)&&discount>10?["Discounts >10% require VP approval for this deal."]:[],
      prerequisites:["Validate relevance for this deal.","Original-case approval does not apply to this deal."],
      unknowns:["Execution outcomes are not evidenced.","Eligibility and approval for this deal are unknown.",...(t.id==="HIST-PUBLIC-REFERENCE"?["Consent to serve as a reference for this prospect is unknown."]:[])]}];
  });
}
function buildDeal(row:Row,date:string,data:Awaited<ReturnType<typeof dataset>>):Deal {
  const account=data.accounts.find(a=>a.account_id===row.account_id)!;
  const timeline=events(row.account_id,date,data);
  const atSnapshot=date===snapshot;
  const sourceIds=timeline.map(e=>e.evidence_ids[0]);
  const employment=data.history.some(h=>h.account_id===row.account_id&&visibleAt(h.mulai,date)&&(!h.selesai||h.selesai>=date));
  const request=timeline.filter(e=>e.type==="BUYER_REQUEST").at(-1);
  const dimensions=[
    {key:"deal_record",label:"Deal record",present:atSnapshot,evidence_ids:atSnapshot?["EVC-"+row.deal_id]:[]},
    {key:"stakeholder_context",label:"Stakeholder context",present:employment&&timeline.length>0,evidence_ids:employment?sourceIds:[]},
    {key:"meaningful_interaction",label:"Meaningful interaction",present:timeline.length>0,evidence_ids:sourceIds},
    {key:"commercial_terms",label:"Commercial terms",present:timeline.some(e=>e.type==="PROPOSAL"||e.type==="DISCOUNT_REQUEST"),evidence_ids:timeline.filter(e=>e.type==="PROPOSAL"||e.type==="DISCOUNT_REQUEST").flatMap(e=>e.evidence_ids)},
    {key:"decision_or_gate_evidence",label:"Decision / purchase gate",present:Boolean(request&&row.account_id==="P04"),evidence_ids:request&&row.account_id==="P04"?request.evidence_ids:[]}
  ];
  return {deal_id:row.deal_id,account_id:row.account_id,name:account.nama,industry:account.industri,city:account.kota,
    owner:atSnapshot?(data.employees.find(e=>e.employee_id===row.owner_id)?.nama??row.owner_id):"Historical owner unknown",owner_id:atSnapshot?row.owner_id:"unknown",
    stage:atSnapshot?row.stage:null,stage_since:atSnapshot?row.stage_sejak:null,
    acv:atSnapshot?Number(row.nilai_tahunan):null,outlets:atSnapshot?Number(row.outlet):null,
    attractiveness:atSnapshot?attractiveness(Number(row.nilai_tahunan),Math.max(...data.deals.filter(d=>/^P/.test(d.account_id)).map(d=>Number(d.nilai_tahunan))),Number(row.outlet)):null,
    urgency:null,readiness:null,readiness_label:null,coverage:dimensions.filter(d=>d.present).length,coverage_dimensions:dimensions,
    blocker:request&&row.account_id==="P04"?"A customer reference was requested before signing; completion is unknown.":null,
    blocker_evidence_ids:request&&row.account_id==="P04"?request.evidence_ids:[],action_count:candidates(date,data).length,
    temporal_scope:"snapshot_only",assessment_status:"jev_unavailable"
  };
}
export async function listDeals(date:string) {
  assertDate(date);const data=await dataset();
  return {scope:scope(date,data.version),deals:data.deals.filter(d=>/^P/.test(d.account_id)&&visibleAt(d.dibuat,date)).map(d=>buildDeal(d,date,data))};
}
export async function context(id:string,date:string):Promise<ContextBundle> {
  assertDate(date);const data=await dataset();const row=data.deals.find(d=>d.deal_id===id&&/^P/.test(d.account_id)&&visibleAt(d.dibuat,date));
  if(!row)throw new Error("Deal not found on this date.");
  const deal=buildDeal(row,date,data),timeline=events(row.account_id,date,data);
  const nodes:GraphNode[]=[{id:deal.deal_id,label:deal.account_id+" · Deal",subtitle:date===snapshot?row.stage:"Historical stage unknown",type:"Deal",verification:"verified",evidence_ids:date===snapshot?["EVC-"+id]:[],x:30,y:160},
    {id:deal.account_id,label:deal.name,subtitle:"Account · "+deal.account_id,type:"Account",verification:"verified",evidence_ids:[],x:270,y:160}];
  const edges:GraphEdge[]=[{id:"deal-account",source:id,target:deal.account_id,type:"FOR_ACCOUNT",verification:"verified",evidence_ids:date===snapshot?["EVC-"+id]:[]}];
  const visible=timeline.slice(-4);
  visible.forEach((event,i)=>{
    const raw=data.interactions.find(r=>r.interaction_id===event.interaction_id)!;
    const y=i*210-160;
    nodes.push({id:event.event_id,label:event.title,type:"Event",subtitle:event.date+" · "+event.type,verification:"verified",evidence_ids:event.evidence_ids,event_id:event.event_id,x:520,y},
      {id:event.interaction_id,label:event.interaction_id,type:"Interaction",subtitle:"Rekaman "+raw.tipe,verification:"verified",evidence_ids:event.evidence_ids,event_id:event.event_id,x:780,y},
      {id:event.evidence_ids[0],label:"Bukti sumber",type:"Evidence",subtitle:event.interaction_id+" · JSONL",verification:"verified",evidence_ids:event.evidence_ids,event_id:event.event_id,x:1040,y});
    edges.push({id:event.event_id+"-account",source:deal.account_id,target:event.event_id,type:"HAS_EVENT",verification:"verified",evidence_ids:event.evidence_ids},
      {id:event.event_id+"-interaction",source:event.event_id,target:event.interaction_id,type:"EXTRACTED_FROM",verification:"verified",evidence_ids:event.evidence_ids},
      {id:event.event_id+"-evidence",source:event.interaction_id,target:event.evidence_ids[0],type:"SUPPORTED_BY",verification:"verified",evidence_ids:event.evidence_ids});
    const personId="PERSON-"+raw.dari;
    if(!nodes.some(n=>n.id===personId))nodes.push({id:personId,label:event.actor,type:"Person",subtitle:"Address on the source",verification:"verified",evidence_ids:event.evidence_ids,x:780,y:y+95});
    edges.push({id:event.event_id+"-actor",source:personId,target:event.interaction_id,type:"SENT_OR_RECORDED",verification:"verified",evidence_ids:event.evidence_ids});
  });
  const dates=[...new Set(data.interactions.filter(i=>i.account_id===row.account_id&&i.tanggal<=snapshot).map(i=>i.tanggal))].sort();
  return {navigation:{previous:dates.filter(d=>d<date).at(-1)??null,next:dates.find(d=>d>date)??(date<snapshot?snapshot:null)},scope:scope(date,data.version),deal,timeline,graph:{nodes,edges,truncated:timeline.length>4,
    unknowns:["Events connect to the deal through its account; emails do not directly attribute a deal_id.","Consent, execution status, and stage history are unverified."]},
    candidates:candidates(date,data),assessment:{status:"jev_unavailable",rubric_version:"readiness-v1.0",explanation:"Attractiveness = 50% relative ACV + 50% strategic footprint (>30). Urgency and readiness await backend assessment and JEV. Coverage measures five information dimensions, not model confidence.",unknowns:["Live JEV is not connected.","The backend defines the five-level readiness mapping."]}};
}
export async function evidence(id:string,date:string):Promise<Evidence> {
  assertDate(date);const data=await dataset();
  if(id.startsWith("EVI-")){
    const row=data.interactions.find(i=>i.interaction_id===id.slice(4)&&visibleAt(i.tanggal,date));
    if(!row)throw new Error("Evidence not found on this date.");
    return {evidence_id:id,source_record_id:row.interaction_id,source_file:"interactions.jsonl",date:row.tanggal,title:row.subjek,excerpt:row.isi,raw_record:row,information_kind:"Observed",account_id:row.account_id,verification:"verified"};
  }
  if(id.startsWith("EVD-")){
    const row=data.decisions.find(i=>i.decision_id===id.slice(4)&&visibleAt(i.tanggal,date));
    if(!row)throw new Error("Evidence not found on this date.");
    return {evidence_id:id,source_record_id:row.decision_id,source_file:"decision_log.csv",date:row.tanggal,title:row.tipe+" · "+row.nilai,excerpt:row.keputusan+". "+row.alasan,raw_record:row,information_kind:"Observed",account_id:row.account_id,verification:"verified"};
  }
  if(id.startsWith("EVC-")&&date===snapshot){
    const row=data.deals.find(d=>d.deal_id===id.slice(4));if(!row)throw new Error("Evidence not found.");
    return {evidence_id:id,source_record_id:row.deal_id,source_file:"crm_deals.csv",date:snapshot,title:"Snapshot CRM · "+row.deal_id,excerpt:"Stage "+display(row.stage)+"; potential ACV IDR "+Number(row.nilai_tahunan).toLocaleString("id-ID")+"; planned footprint "+row.outlet+" outlets.",raw_record:row,information_kind:"Observed",account_id:row.account_id,verification:"verified"};
  }
  throw new Error("Evidence not found on this date.");
}
