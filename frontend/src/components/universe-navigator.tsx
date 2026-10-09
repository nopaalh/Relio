"use client";
import {useEffect,useRef,useState} from 'react';
import {CalendarDays,ChevronLeft,ChevronRight,Search,Expand,Minimize2,History,LoaderCircle} from 'lucide-react';
import {request} from '@/lib/client';
import {dateDisplay} from '@/lib/display';
import {validDate} from '@/lib/domain';
const latest='2026-10-01',day=86400000;
export default function UniverseNavigator({dealId,date,onChange}:{dealId:string;date:string;onChange:(date:string)=>void}){
 const [dates,setDates]=useState<string[]>([]),[query,setQuery]=useState(''),[draft,setDraft]=useState(date),[expanded,setExpanded]=useState(false),[error,setError]=useState('');
 const timer=useRef<ReturnType<typeof setTimeout>|null>(null),rail=useRef<HTMLDivElement>(null);
 useEffect(()=>{const controller=new AbortController();setDates([]);setError('');
 // Use the existing timeline endpoint once for the calendar index. Only dates enter UI state.
 request<{events:{date:string}[]}>('/api/deals/'+encodeURIComponent(dealId)+'/timeline?as_of='+latest,{signal:controller.signal}).then(r=>setDates([...new Set([...r.events.map(e=>e.date),latest])].sort())).catch(e=>{if(!controller.signal.aborted)setError(e.message)});return()=>controller.abort();},[dealId]);
 useEffect(()=>{setDraft(date);if(timer.current)clearTimeout(timer.current);rail.current?.querySelector('[aria-current="date"]')?.scrollIntoView({block:'nearest',inline:'nearest'});},[date,dates]);
 useEffect(()=>()=>{if(timer.current)clearTimeout(timer.current)},[]);
 const previous=dates.filter(d=>d<date).at(-1),next=dates.find(d=>d>date),shown=dates.filter(d=>(d+' '+dateDisplay(d)).toLowerCase().includes(query.toLowerCase()));
 const minimum=Date.parse(dates[0]??date)/day,maximum=Date.parse(latest)/day;
 function commit(value:string){if(timer.current)clearTimeout(timer.current);if(validDate(value)&&value>='2024-01-01'&&value<=latest&&value!==date)onChange(value)}
 function scrub(value:string){setDraft(value);if(timer.current)clearTimeout(timer.current);timer.current=setTimeout(()=>commit(value),350)}
 return <section className={'universe-navigator '+(expanded?'expanded':'')} aria-label="Universe history"><div className="navigator-heading"><div><History size={17}/><strong>Travel through time</strong><span>One universe, seen at a different moment.</span></div><button className="icon-button" aria-label={expanded?'Collapse history':'Expand history'} aria-expanded={expanded} onClick={()=>setExpanded(v=>!v)}>{expanded?<Minimize2 size={17}/>:<Expand size={17}/>}</button></div>
 <div className="date-rail-controls"><button className="icon-button" aria-label="Previous universe date" disabled={!previous} onClick={()=>previous&&commit(previous)}><ChevronLeft size={18}/></button><div ref={rail} className="date-rail" aria-label="Available universe dates">{shown.map(d=><button key={d} aria-current={d===date?'date':undefined} className={d===date?'active':''} onClick={()=>commit(d)}><CalendarDays size={12}/>{dateDisplay(d)}{d===latest&&<small>Snapshot</small>}</button>)}{!dates.length&&!error&&<span><LoaderCircle className="spin" size={14}/>Loading dates…</span>}{dates.length>0&&shown.length===0&&<span>No matching date. Try a month or YYYY-MM-DD.</span>}</div><button className="icon-button" aria-label="Next universe date" disabled={!next} onClick={()=>next&&commit(next)}><ChevronRight size={18}/></button></div>
 <div className="history-tools"><label className="history-search"><Search size={15}/><input type="search" aria-label="Search universe dates" placeholder="Search dates · Sep or 2026-09-22" value={query} onChange={e=>setQuery(e.target.value)} onKeyDown={e=>{if(e.key==='Enter'){e.preventDefault();if(validDate(query))commit(query);else if(shown.length===1)commit(shown[0])}}}/></label><div className="history-scrubber"><label htmlFor="universe-scrubber">Scrub history <output>{dateDisplay(draft)}</output></label><input id="universe-scrubber" type="range" aria-label="Scrub universe history" min={minimum} max={maximum} step={1} value={Date.parse(draft)/day} disabled={!dates.length} onChange={e=>scrub(new Date(Number(e.target.value)*day).toISOString().slice(0,10))} onPointerUp={()=>commit(draft)} onKeyUp={e=>{if(['ArrowLeft','ArrowRight','Home','End'].includes(e.key))commit(draft)}}/></div></div>{error&&<p className="small muted" role="status">Date index unavailable. Use the date picker above.</p>}
 </section>;
}
