"use client";
import { useEffect, useState } from "react";
import { FileText, ExternalLink, ShieldCheck, Copy, Check, Info } from "lucide-react";
import { api } from "@/lib/client";
import { display } from "@/lib/display";
import type { Evidence, SnapshotMeta } from "@/lib/contracts";
export function Sources({ids,onOpen}:{ids:string[];onOpen:(id:string)=>void}){
  return <div className="source-links">{ids.map(id=><button key={id} onClick={()=>onOpen(id)} className="source-link"><FileText size={13}/>{id.replace(/^EV[IDC]-/,"")}<ExternalLink size={11}/></button>)}</div>;
}
export default function EvidencePanel({id,date,expectedMeta}:{id:string|null;date:string;expectedMeta?:SnapshotMeta}){
  const [record,setRecord]=useState<Evidence|null>(null),[error,setError]=useState(""),[retry,setRetry]=useState(0),[copied,setCopied]=useState(false);
  useEffect(()=>{
    setRecord(null);setError("");setCopied(false);if(!id)return;
    const controller=new AbortController();
    api.evidence(id,date,controller.signal,expectedMeta).then(data=>{if(!controller.signal.aborted)setRecord(data)}).catch(e=>{if(!controller.signal.aborted)setError(e.message)});
    return ()=>controller.abort();
  },[id,date,retry,expectedMeta]);
  if(!id)return <div className="evidence-empty"><div className="empty-icon"><FileText size={25}/></div><h3>Follow the evidence</h3><p>Select an event, planet, or citation to inspect its authorized source excerpt.</p><div className="mini-path"><span>Context</span><i>→</i><span>Source</span><i>→</i><span>Evidence</span></div></div>;
  if(error)return <div role="alert" className="empty"><p>{error}</p><button className="button secondary" onClick={()=>setRetry(n=>n+1)}>Try again</button></div>;
  if(!record)return <div className="panel-loading" role="status"><div className="skeleton line"/><div className="skeleton block"/><p>Opening source…</p></div>;
  const source=record.source,accounts=source?source.account_ids.join(", "):record.account_id;
  return <div className="evidence-content"><div className="evidence-kicker"><span className={"badge "+(record.verification==="verified"?"teal":"neutral")}><ShieldCheck size={12}/>{record.information_kind}</span><span className="mono">{record.source_record_id}</span></div>
    <h3>{display(record.title)}</h3><dl className="source-meta"><div><dt>Source</dt><dd className="mono">{record.source_file}</dd></div><div><dt>Date</dt><dd>{record.date??"Unknown / undated"}</dd></div><div><dt>Source accounts</dt><dd>{accounts||"Account scope unknown"}</dd></div><div><dt>Verification</dt><dd>{record.verification||"Unknown"}</dd></div>
      {source&&<><div><dt>Scope</dt><dd>{source.scope_kind} · {source.deal_ids.length?source.deal_ids.join(", "):"No direct deal linkage"}</dd></div><div><dt>Source field</dt><dd className="mono">{source.source_field}</dd></div><div><dt>Locator kind</dt><dd>{source.record_id_kind}</dd></div><div><dt>Temporal basis</dt><dd>{source.temporal_basis}</dd></div><div><dt>Source timestamp</dt><dd>{source.source_timestamp??"Not supplied; no clock time inferred"}</dd></div></>}
    </dl>
    <blockquote>{record.excerpt??"Excerpt unavailable; no source content can be inferred."}</blockquote>
    <p className="muted small">{source?"Authorized source field/span only—not the full row or thread. Requests, approvals and execution remain distinct.":record.source_file==="crm_deals.csv"?"Local summary of source fields. Inspect the local record below.":"Local source text. Requests, approvals and execution are distinct states."}</p>
    {source&&<details className="raw-source"><summary>Scoped provenance</summary><dl><div><dt>Evidence ID</dt><dd className="mono">{source.evidence_id}</dd></div><div><dt>Source checksum</dt><dd className="mono">{source.source_checksum}</dd></div><div><dt>Span</dt><dd>{source.span?source.span.start+"–"+source.span.end+" Unicode code points [start, end)":"Scalar field; no span supplied"}</dd></div>{source.span&&<div><dt>Field checksum</dt><dd className="mono">{source.span.field_checksum}</dd></div>}{Object.entries(source.source_key).map(([key,value])=><div key={key}><dt className="mono">{key}</dt><dd>{value}</dd></div>)}<div><dt>Events</dt><dd>{source.event_ids.join(", ")||"None supplied"}</dd></div><div><dt>Edges</dt><dd>{source.edge_ids.join(", ")||"None supplied"}</dd></div><div><dt>Occurrences</dt><dd>{source.occurrence_ids.join(", ")||"None supplied"}</dd></div></dl></details>}
    {!source&&record.raw_record&&<details className="raw-source"><summary>Full source record</summary><dl>{Object.entries(record.raw_record).map(([key,value])=><div key={key}><dt className="mono">{key}</dt><dd>{value||"Not recorded"}</dd></div>)}</dl></details>}
    {record.meta&&<details className="raw-source"><summary>Snapshot metadata & unknowns</summary><dl><div><dt>Dataset / contract</dt><dd>{record.meta.dataset_version} / {record.meta.contract_version}</dd></div><div><dt>Context</dt><dd>{record.meta.context_id}</dd></div><div><dt>As of / calendar</dt><dd>{record.meta.as_of} / {record.meta.calendar_zone}</dd></div><div><dt>Data state</dt><dd>{record.meta.data_state}</dd></div></dl><ul>{record.meta.limitations.map((text,i)=><li key={i}>{text}</li>)}{record.meta.unknowns.map((u,i)=><li key={i}>{u.path}: {u.reason}</li>)}</ul></details>}
    {record.information_kind==="Unknown"&&<p className="small muted">Information kind was not supplied; verification is shown separately and is not an approval.</p>}
    <button className="button secondary full" disabled={record.excerpt===null} onClick={async()=>{try{await navigator.clipboard.writeText(record.source_record_id+" · "+record.source_file+(source?" · "+source.source_field:"")+" · "+(record.date??"Undated")+"\n"+record.excerpt);setCopied(true);}catch{setError("Automatic copying is unavailable in this browser.");}}}>{copied?<Check size={15}/>:<Copy size={15}/>} {copied?"Quote copied":"Copy quote"}</button>
    <div className="callout neutral"><Info size={16}/><span>Evidence from {accounts||"the source scope"} does not approve a different deal.</span></div>
  </div>;
}
