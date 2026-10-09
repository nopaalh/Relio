"use client";
import { useEffect, useRef, useId } from "react";
import { X } from "lucide-react";
export default function Dialog({title,onClose,children,wide=false}:{title:string;onClose:()=>void;children:React.ReactNode;wide?:boolean}){
  const ref=useRef<HTMLDialogElement>(null),previousFocus=useRef<HTMLElement|null>(null),id=useId();
  useEffect(()=>{const element=ref.current;if(!element)return;if(!previousFocus.current&&!element.contains(document.activeElement))previousFocus.current=document.activeElement as HTMLElement|null;if(!element.open)element.showModal();return ()=>{element.close();previousFocus.current?.focus({preventScroll:true});};},[]);
  return <dialog ref={ref} aria-labelledby={id} className={wide?"modal wide":"modal"} onCancel={onClose} onClick={e=>{if(e.target===ref.current)onClose();}}>
    <header><div><span className="eyebrow">RELIO WORKSPACE</span><h2 id={id}>{title}</h2></div><button className="icon-button" aria-label="Close panel" onClick={onClose}><X size={20}/></button></header>
    <div className="modal-body">{children}</div>
  </dialog>;
}
