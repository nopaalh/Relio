"use client";
import { useEffect, useState } from "react";
import { FileText, ExternalLink, ShieldCheck, Copy, Check, Info } from "lucide-react";
import { api } from "@/lib/client";
import { display } from "@/lib/display";
import type { Evidence } from "@/lib/contracts";
export function Sources({ids,onOpen}:{ids:string[];onOpen:(id:string)=>void}){
  return <div className="source-links">{ids.map(id=><button key={id} onClick={()=>onOpen(id)} className="source-link"><FileText size={13}/>{id.replace(/^EV[IDC]-/,"")}<ExternalLink size={11}/></button>)}</div>;
}
export default function EvidencePanel({id,date}:{id:string|null;date:string}){
  const [record,setRecord]=useState<Evidence|null>(null),[error,setError]=useState(""),[retry,setRetry]=useState(0),[copied,setCopied]=useState(false);
  useEffect(()=>{setRecord(null);setError("");setCopied(false);if(!id)return;const controller=new AbortController();api.evidence(id,date,controller.signal).then(setRecord).catch(e=>{if(!controller.signal.aborted)setError(e.message);});return ()=>controller.abort();},[id,date,retry]);
  if(!id)return <div className="evidence-empty"><div className="empty-icon"><FileText size={25}/></div><h3>Follow the evidence</h3><p>Select an event, planet, or citation to inspect its original record.</p><div className="mini-path"><span>Context</span><i>→</i><span>Source</span><i>→</i><span>Evidence</span></div></div>;
  if(error)return <div role="alert" className="empty"><p>{error}</p><button className="button secondary" onClick={()=>setRetry(n=>n+1)}>Try again</button></div>;
  if(!record)return <div className="panel-loading" role="status"><div className="skeleton line"/><div className="skeleton block"/><p>Opening source…</p></div>;
  return <div className="evidence-content"><div className="evidence-kicker"><span className="badge teal"><ShieldCheck size={12}/>Observed</span><span className="mono">{record.source_record_id}</span></div>
    <h3>{display(record.title)}</h3><dl className="source-meta"><div><dt>Source</dt><dd className="mono">{record.source_file}</dd></div><div><dt>Date</dt><dd>{record.date??"Unknown"}</dd></div><div><dt>Source account</dt><dd>{record.account_id||"Internal"}</dd></div></dl>
    <blockquote>{record.excerpt}</blockquote><p className="muted small">{record.source_file==="crm_deals.csv"?"Summary of source fields. Inspect the raw record below.":"Original source text (Indonesian). Requests, approvals, and execution are distinct states."}</p>
    <details className="raw-source"><summary>Full source record</summary><dl>{Object.entries(record.raw_record).map(([key,value])=><div key={key}><dt className="mono">{key}</dt><dd>{value||"Not recorded"}</dd></div>)}</dl></details>
    <button className="button secondary full" onClick={async()=>{try{await navigator.clipboard.writeText(record.source_record_id+" · "+record.source_file+" · "+record.date+"\n"+record.excerpt);setCopied(true);}catch{setError("Automatic copying is unavailable in this browser.");}}}>{copied?<Check size={15}/>:<Copy size={15}/>} {copied?"Quote copied":"Copy quote"}</button>
    <div className="callout neutral"><Info size={16}/><span>Evidence from {record.account_id||"the source case"} does not approve a different deal.</span></div>
  </div>;
}
