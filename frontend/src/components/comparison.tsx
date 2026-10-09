"use client";
import { useEffect, useRef, useState } from "react";
import { ArrowRight, Check, GitCompareArrows, Info, AlertTriangle, LoaderCircle, ShieldCheck } from "lucide-react";
import type { ActionCandidate, Comparison } from "@/lib/contracts";
import { api } from "@/lib/client";
import { fingerprint, validateSelection, validChoice } from "@/lib/domain";
import { Sources } from "./evidence";
export default function Compare({candidates,dealId,date,version,onSource}:{candidates:ActionCandidate[];dealId:string;date:string;version:string;onSource:(id:string)=>void}){
  const [ids,setIds]=useState<string[]>([]),[result,setResult]=useState<Comparison|null>(null),[busy,setBusy]=useState(false),[error,setError]=useState(""),[stale,setStale]=useState(false);
  const controller=useRef<AbortController|null>(null),runId=useRef(0),errorRef=useRef<HTMLDivElement>(null);
  useEffect(()=>{controller.current?.abort();runId.current++;setIds([]);setResult(null);setBusy(false);setError("");setStale(false);return ()=>controller.current?.abort();},[dealId,date,version]);
  function toggle(id:string){controller.current?.abort();runId.current++;setBusy(false);setError("");if(result)setStale(true);setResult(null);setIds(prev=>prev.includes(id)?prev.filter(v=>v!==id):[...prev,id]);}
  async function compare(){
    const validation=validateSelection(ids,candidates.map(c=>c.action_id));if(validation){setError(validation);setTimeout(()=>errorRef.current?.focus(),0);return;}
    const requestKey=fingerprint(dealId,date,ids,version),token=++runId.current;
    controller.current?.abort();const abort=new AbortController();controller.current=abort;
    setBusy(true);setError("");setResult(null);setStale(false);
    try{const data=await api.compare(dealId,date,ids,abort.signal);if(token===runId.current&&requestKey===fingerprint(dealId,date,ids,version)){
      if(data.deal_id!==dealId||data.as_of!==date||data.items.length!==ids.length||data.items.some(item=>!ids.includes(item.action_id))||new Set(data.items.map(i=>i.action_id)).size!==ids.length)throw new Error("Response does not match the scope or selected set.");
      setResult(data);
    }}catch(e){if(!abort.signal.aborted){setError(e instanceof Error?e.message:"Assessment unavailable.");setTimeout(()=>errorRef.current?.focus(),0);}}
    finally{if(token===runId.current)setBusy(false);}
  }
  const choiceValid=result?validChoice(result.items.map(i=>i.jev_choice_preference)):false;
  const scoresComplete=result?.items.every(i=>i.suitability_score_100!==null&&Number.isFinite(i.suitability_score_100)&&i.suitability_score_100>=0&&i.suitability_score_100<=100);
  const scoreWinner=result?.items.find(i=>i.rank===1)?.action_id;
  const choiceWinner=choiceValid?[...result!.items].sort((a,b)=>(b.jev_choice_preference??0)-(a.jev_choice_preference??0))[0]?.action_id:null;
  const disagreement=scoreWinner&&choiceWinner&&scoreWinner!==choiceWinner;
  return <section className="comparison"><div className="section-heading"><div><span className="eyebrow">NEXT MOVE</span><h2>Compare approaches with evidence.</h2><p>Choose 2–4 sourced historical approaches. Compare their suitability for this deal.</p></div><span className="badge blue"><GitCompareArrows size={13}/>One deal · 2–4 options</span></div>
    <div className="callout neutral"><Info size={17}/><span>These precedents belong to other cases. Relevance, outcomes, and approval for this deal still require validation.</span></div>
    {candidates.length<2?<div className="empty"><h3>Insufficient historical evidence</h3><p>Found {candidates.length} sourced approaches. Comparison requires at least two.</p></div>:<div className="candidate-grid">{candidates.map(candidate=>{
      const selected=ids.includes(candidate.action_id),limited=ids.length===4&&!selected;
      return <article className={"candidate "+(selected?"is-selected":"")} key={candidate.action_id}>
        <label className="candidate-label"><input type="checkbox" checked={selected} disabled={limited||busy} onChange={()=>toggle(candidate.action_id)}/><span className="check-ui">{selected&&<Check size={14}/>}</span><span><span className="eyebrow">HISTORICAL APPROACH</span><strong>{candidate.title}</strong></span></label><p>{candidate.description}</p><span className="badge amber">Needs validation</span>
        {candidate.occurrences.map(o=><div className="precedent" key={o.occurrence_id}><span className="small muted">PRECEDENT · {o.date}</span><strong>{o.account_name}</strong><div className="precedent-status"><ShieldCheck size={13}/>{o.status} in the original case · Outcome {o.outcome_observed??"unknown"}</div><Sources ids={o.evidence_ids} onOpen={onSource}/></div>)}
        {candidate.policy_flags.map(flag=><div className="policy-note" key={flag}><AlertTriangle size={14}/><span>{flag}</span></div>)}
        <details><summary>Prerequisites & limitations</summary><ul>{[...candidate.prerequisites,...candidate.unknowns].map(t=><li key={t}>{t}</li>)}</ul></details>
      </article>;
    })}</div>}
    <div className="compare-toolbar"><div aria-live="polite"><strong>{ids.length} of 4 selected</strong><p className="small muted">{ids.length<2?"Select at least two approaches to compare.":ids.length===4?"Four selected. Deselect an approach to replace it.":"Suitability is not closing probability."}</p></div><button className="button primary" disabled={ids.length<2||busy} onClick={()=>void compare()}>{busy?<LoaderCircle className="spin" size={17}/>:<GitCompareArrows size={17}/>} {busy?"Assessing approaches…":"Compare approaches"}<ArrowRight size={16}/></button></div>
    {stale&&<p className="callout neutral" role="status">Selection changed. Run again to assess this set.</p>}
    {error&&<div ref={errorRef} tabIndex={-1} className="callout danger" role="alert">{error}</div>}
    {result&&<div className="comparison-result" aria-live="polite"><div className="section-heading"><div><h3>Comparison results</h3><p>{result.rubric_version} · {date}</p></div><span className={"badge "+(result.assessment_status==="jev_unavailable"?"amber":"blue")}>{result.assessment_status==="jev_unavailable"?"JEV not connected":result.assessment_status}</span></div>
      {disagreement&&<div className="callout amber"><AlertTriangle size={17}/>Review needed: Score ranking differs from Choice preference. They are shown separately.</div>}
      <div className="result-grid">{result.items.map(item=><article className="result-card" key={item.action_id}><h4>{candidates.find(c=>c.action_id===item.action_id)?.title}</h4><div className="score-value">{item.suitability_score_100??"—"}<span>/100</span></div><p>Suitability per approach</p><dl><div><dt>Rank</dt><dd>{scoresComplete?item.rank??"—":"Unavailable"}</dd></div><div><dt>JEV Score / 4</dt><dd>{item.jev_raw_score??"—"}</dd></div><div><dt>Choice · selected set</dt><dd>{choiceValid?(100*item.jev_choice_preference!).toFixed(1)+"%":"Unavailable"}</dd></div></dl><p className="small muted">{item.unknowns.join(" ")}</p><Sources ids={item.evidence_ids} onOpen={onSource}/></article>)}</div>
      <p className="callout neutral"><Info size={17}/>{result.warning} {result.assessment_status==="jev_unavailable"&&"No scores or rankings are generated while the service is unavailable."}</p>
    </div>}
  </section>;
}
