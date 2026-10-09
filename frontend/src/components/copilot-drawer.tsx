"use client";
import {useEffect,useRef,useId} from 'react';
import {MessageSquare,X} from 'lucide-react';
import Copilot from './copilot';
import type {ChatEntity} from './chat-context';
export default function CopilotDrawer({date,dealId,entity,localMode,onClose,onSource}:{date:string;dealId:string|null;entity?:ChatEntity|null;localMode:boolean;onClose:()=>void;onSource:(id:string,date?:string)=>void}){
 const ref=useRef<HTMLDialogElement>(null),previous=useRef<HTMLElement|null>(null),id=useId();
 useEffect(()=>{const panel=ref.current;if(!panel)return;previous.current=document.activeElement as HTMLElement;
 const mobile=window.matchMedia('(max-width: 600px)').matches;if(mobile)panel.showModal();else panel.show();
 panel.querySelector<HTMLTextAreaElement>('textarea')?.focus({preventScroll:true});
 const key=(e:KeyboardEvent)=>{if(e.key==='Escape'&&!document.querySelector('.modal[open]')){e.preventDefault();onClose()}};
 window.addEventListener('keydown',key);return()=>{window.removeEventListener('keydown',key);panel.close();const fallback=document.querySelector<HTMLElement>('[data-copilot-trigger]');(previous.current?.isConnected?previous.current:fallback)?.focus({preventScroll:true})};
 // onClose is read through native button/cancel; scope changes keep one chat mounted.
 // eslint-disable-next-line react-hooks/exhaustive-deps
 },[]);
 return <dialog ref={ref} aria-labelledby={id} aria-modal={undefined} className="copilot-drawer" onCancel={e=>{e.preventDefault();onClose()}}><header><div><span className="drawer-spark"><MessageSquare size={19}/></span><div><span className="eyebrow">RELIO</span><h2 id={id}>Copilot</h2></div></div><button className="icon-button" aria-label="Close Copilot" onClick={onClose}><X size={19}/></button></header><Copilot date={date} dealId={dealId} entity={entity} localMode={localMode} onSource={onSource}/></dialog>;
}
