"use client";
import {useId,useRef,useState} from 'react';
import {ArrowUp,MessageSquare,Bot,Info,LoaderCircle,Orbit} from 'lucide-react';
import {api,ApiError} from '@/lib/client';
import {useChat,type ChatEntity} from './chat-context';
import {Sources} from './evidence';
import {display,dateDisplay} from '@/lib/display';
export default function Copilot({date,dealId,entity,localMode=false,onSource}:{date:string;dealId:string|null;entity?:ChatEntity|null;localMode?:boolean;onSource:(id:string,date?:string)=>void}){
 const {messages,setMessages}=useChat(),[question,setQuestion]=useState(''),[busy,setBusy]=useState(false),ref=useRef<HTMLTextAreaElement>(null),inputId=useId();
 async function ask(value:string){
  if(!value.trim()||busy)return;const id=crypto.randomUUID(),scope={date,dealId,entity:entity??undefined};setBusy(true);setQuestion('');setMessages(prev=>[...prev,{id,question:value,...scope}]);
  // Keep the existing API contract. Include the selected source-backed entity ID in the query.
  const prefix=entity?('Context entity: '+entity.id+' ('+entity.type+': '+entity.label+'). ').slice(0,350):'';
  try{const response=await api.ask(prefix+'Answer in English and cite available sources. '+value,date,dealId);setMessages(prev=>prev.map(m=>m.id===id?{...m,response}:m))}
  catch(e){setMessages(prev=>prev.map(m=>m.id===id?{...m,error:e instanceof Error?e.message:'Service unavailable.',unavailable:e instanceof ApiError&&e.status===503}:m))}
  finally{setBusy(false);ref.current?.focus()}
 }
 const suggestions=entity?['What does this entity tell us?','Which connections matter here?','What evidence is still missing?']:dealId?['What blockers are supported by evidence?','Who is involved in this decision?','What is still unknown about this deal?']:['Which deals need closer review?','Which discounts have VP approval?','Who last contacted a visible account?'];
 return <div className="copilot"><div className="copilot-intro"><div className="ai-icon"><MessageSquare size={24}/></div><h3>{entity?'Ask about this planet':'Ask about your deals'}</h3><p>Questions use the deal and date shown below.</p><div className="scope-pill">{dealId??'Workspace scope'} · {dateDisplay(date)}</div>{localMode&&<p className="copilot-service-note">Local demo: the answer provider is not connected. Questions can be submitted to check the unavailable state.</p>}</div>
 {entity&&<div className="copilot-entity"><Orbit size={19}/><div><small>{entity.type}</small><strong>{display(entity.label)}</strong><code>{entity.id}</code></div></div>}
 <div className="suggestions">{suggestions.map(q=><button key={q} onClick={()=>{setQuestion(q);ref.current?.focus()}}>{q}<ArrowUp size={14}/></button>)}</div>
 <div className="messages" aria-live="polite">{messages.map(m=><article key={m.id} className="message"><div className="user-message"><span className="mono small">{m.dealId??'Workspace scope'} · {m.date}</span>{m.entity&&<span className="message-entity">{m.entity.type} · {display(m.entity.label)} · {m.entity.id}</span>}<p>{m.question}</p></div><div className="assistant-message"><Bot size={20}/><div>{!m.response&&!m.error?<p><LoaderCircle className="spin" size={15}/>Checking context…</p>:<><p>{m.error??m.response?.answer}</p>{(m.unavailable||m.response?.status==='unavailable')&&<span className="badge neutral">Service not connected</span>}<Sources ids={m.response?.evidence_ids??[]} onOpen={id=>onSource(id,m.date)}/></>}</div></div></article>)}</div>
 <form onSubmit={e=>{e.preventDefault();void ask(question)}} className="prompt-box"><label className="sr-only" htmlFor={inputId}>Question for Copilot</label><textarea ref={ref} id={inputId} placeholder={entity?'Ask about this planet…':'Ask about your deal universe…'} rows={2} maxLength={1500} value={question} onChange={e=>setQuestion(e.target.value)} onKeyDown={e=>{if(e.key==='Enter'&&!e.shiftKey){e.preventDefault();void ask(question)}}}/><button className="button primary icon-button" aria-label="Send question" disabled={busy||!question.trim()}>{busy?<LoaderCircle className="spin" size={18}/>:<ArrowUp size={20}/>}</button></form>
 <p className="copilot-footnote"><Info size={12}/>Check cited sources before making a decision.</p></div>;
}
