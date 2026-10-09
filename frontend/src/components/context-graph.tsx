"use client";
import { useEffect, useMemo, useRef, useState } from "react";
import { ReactFlow, ReactFlowProvider, ViewportPortal, getViewportForBounds, BaseEdge, EdgeLabelRenderer, Handle, Position, useReactFlow, useNodesInitialized, type NodeProps, type Node, type Edge, type EdgeProps } from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { Network, FileText, Building2, UserRound, CalendarDays, BriefcaseBusiness, ZoomIn, ZoomOut, Maximize2, LocateFixed, List, ArrowLeft, ShieldCheck, GitBranch, Orbit, Sparkles, Search, Ellipsis } from "lucide-react";
import type { GraphNode, GraphEdge, TimelineEvent } from "@/lib/contracts";
import { neighborhood, orbitalLayout, solarSystemLayout, nodeEventIds, eventNode, linkedEvent, type GraphSelection } from "@/lib/universe";
import { display, dateDisplay } from "@/lib/display";
const icons:Record<string,typeof Network>={Deal:BriefcaseBusiness,Account:Building2,Person:UserRound,Event:CalendarDays,Interaction:FileText,Evidence:ShieldCheck,ActionOccurrence:GitBranch,Decision:GitBranch,Contract:FileText,UsageAggregate:Network};
type Data = { label:string;original:GraphNode;highlighted:boolean;date:string;dated:boolean };
type PlanetNode=Node<Data>;
type OrbitEdge=Edge<{original:GraphEdge;sourceRadius:number;targetRadius:number;showLabel:boolean}>;
type Props={nodes:GraphNode[];edges:GraphEdge[];selectedEvent:string|null;selection:GraphSelection|null;asOf:string;events:TimelineEvent[];onAsk:()=>void;onSelect:(ids:string[],eventId?:string)=>void;onInspect:(selection:GraphSelection|null)=>void};
function EntityNode({data,selected}:NodeProps<PlanetNode>){
  const node=data.original,Icon=icons[node.type]??Network;
  return <div className={"planet-node planet-"+node.type.toLowerCase()+" "+(selected?"is-focused":"")+" "+(data.highlighted?"event-highlighted":"")} title={display(node.label)+" · "+node.type}>
    <time className="planet-date" dateTime={data.date}><CalendarDays size={11}/><span>{data.dated?(node.type==="Event"?"Event":"Source"):"As of"}</span>{dateDisplay(data.date)}</time><div className="planet-orb"><span className="planet-orb-ring"/><span className="planet-orb-shine"/><Icon size={24} strokeWidth={1.6}/><span className="planet-satellite"/><Handle type="target" position={Position.Left} className="planet-handle" isConnectable={false}/><Handle type="source" position={Position.Right} className="planet-handle" isConnectable={false}/></div>
    <div className="planet-caption"><span className="planet-type">{node.type}</span><strong>{display(node.label)}</strong></div>
  </div>;
}
function OrbitalEdge({id,sourceX,sourceY,targetX,targetY,selected,data,markerEnd}:EdgeProps<OrbitEdge>){
  const dx=targetX-sourceX,dy=targetY-sourceY,distance=Math.max(1,Math.hypot(dx,dy)),ux=dx/distance,uy=dy/distance;
  const sx=sourceX+ux*(data?.sourceRadius??48),sy=sourceY+uy*(data?.sourceRadius??48);
  const tx=targetX-ux*(data?.targetRadius??48),ty=targetY-uy*(data?.targetRadius??48);
  const bend=Math.min(30,distance*.08),cx=(sx+tx)/2-uy*bend,cy=(sy+ty)/2+ux*bend;
  const inferred=data?.original.verification!=="verified";
  return <><BaseEdge id={id} path={"M "+sx+" "+sy+" Q "+cx+" "+cy+" "+tx+" "+ty} markerEnd={markerEnd} interactionWidth={24} style={{stroke:selected?"#D2BDFF":inferred?"#9DB3FF":"#8797C4",strokeWidth:selected?2.4:1.5,strokeOpacity:selected?1:.65,strokeDasharray:inferred?"6 5":undefined}}/>
    {(selected||data?.showLabel)&&<EdgeLabelRenderer><span className="orbital-edge-label" style={{transform:"translate(-50%, -50%) translate("+cx+"px,"+cy+"px)"}}>{data?.original.type.replaceAll("_"," ")}</span></EdgeLabelRenderer>}</>;
}
const nodeTypes={planet:EntityNode},edgeTypes={orbital:OrbitalEdge};
const systemPadding={top:'150px',bottom:'100px',left:'32px',right:'32px'} as const;
const systemViewport=(bounds:{x:number;y:number;width:number;height:number},width:number,height:number)=>getViewportForBounds(bounds,width,height,.18,.85,systemPadding);
function Canvas({nodes:original,edges:originalEdges,selectedEvent,selection,asOf,events,onAsk,onSelect,onInspect}:Props){
  const ref=useRef<HTMLDivElement>(null),flow=useReactFlow();
  const [query,setQuery]=useState(""),[allContext,setAllContext]=useState(false),[list,setList]=useState(false),[hoverEdge,setHoverEdge]=useState<string|null>(null),[reduced,setReduced]=useState(false);
  const currentNode=selection?.kind==="node"?selection.id:null,currentEdge=selection?.kind==="edge"?selection.id:null;
  useEffect(()=>{const media=window.matchMedia("(prefers-reduced-motion: reduce)");setReduced(media.matches);const listener=()=>setReduced(media.matches);media.addEventListener("change",listener);return ()=>media.removeEventListener("change",listener);},[]);
  const visible=useMemo(()=>{
    if(currentNode)return neighborhood(currentNode,original,originalEdges);
    if(allContext||!selectedEvent)return new Set(original.map(n=>n.id));
    const eventNodes=original.filter(n=>nodeEventIds(n).includes(selectedEvent));
    const senders=originalEdges.filter(e=>eventNodes.some(n=>n.id===e.target)&&e.type==="SENT_OR_RECORDED").map(e=>e.source);
    return new Set([...original.filter(n=>n.type==="Deal"||n.type==="Account").map(n=>n.id),...eventNodes.map(n=>n.id),...senders]);
  },[currentNode,allContext,selectedEvent,original,originalEdges]);
  const subset=useMemo(()=>original.filter(n=>visible.has(n.id)),[original,visible]);
  const layout=useMemo(()=>solarSystemLayout(subset,events),[subset,events]);
  const positions=useMemo(()=>currentNode?orbitalLayout(subset,currentNode):layout.positions,[subset,currentNode,layout]);
  const systems=currentNode?[]:layout.systems;
  const bounds=useMemo(()=>({x:0,y:0,width:Math.max(720,...layout.systems.map(s=>s.x+s.size)),height:Math.max(720,...layout.systems.map(s=>s.y+s.size))}),[layout]);
  const radius=(node:GraphNode)=>node.type==="Deal"?58:node.type==="Account"?52:46;
  const nodes=useMemo<PlanetNode[]>(()=>subset.map(n=>{
      const refs=nodeEventIds(n),dates=[...new Set(events.filter(e=>refs.includes(e.event_id)).map(e=>e.date))],dated=dates.length===1;
      return {id:n.id,type:"planet",position:positions.get(n.id)??{x:0,y:0},selected:n.id===currentNode,data:{label:display(n.label),original:n,highlighted:Boolean(selectedEvent&&refs.includes(selectedEvent)),date:dated?dates[0]:asOf,dated},ariaLabel:n.type+": "+display(n.label),draggable:false,width:170,height:200,style:{width:156,height:164}};
    }),[subset,positions,currentNode,selectedEvent,events,asOf]);
  const edges=useMemo<OrbitEdge[]>(()=>originalEdges.filter(e=>visible.has(e.source)&&visible.has(e.target)).map(e=>({id:e.id,source:e.source,target:e.target,type:"orbital",selected:e.id===currentEdge,focusable:true,data:{original:e,sourceRadius:radius(original.find(n=>n.id===e.source)!),targetRadius:radius(original.find(n=>n.id===e.target)!),showLabel:hoverEdge===e.id},ariaLabel:e.type+" from "+e.source+" to "+e.target})),[originalEdges,visible,currentEdge,hoverEdge,original]);
  const camera=useRef(0), initialized=useNodesInitialized({includeHiddenNodes:true});
  const cameraState=useRef({currentNode,positions,bounds});cameraState.current={currentNode,positions,bounds};
  useEffect(()=>{
    const el=ref.current;if(!el)return;
    let width=0;const observer=new ResizeObserver(entries=>{const size=entries[0]?.contentRect;if(!size||size.width===width)return;width=size.width;if(size.width>0&&size.height>0)requestAnimationFrame(()=>{const {currentNode,positions,bounds}=cameraState.current;const position=currentNode?positions.get(currentNode):null;if(position)void flow.setCenter(position.x+85,position.y+92,{zoom:size.width<430?0.8:1.02,duration:0});else void flow.setViewport(systemViewport(bounds,size.width,size.height),{duration:0});});});
    observer.observe(el);return ()=>observer.disconnect();
  },[flow]);
  const signature=[...visible].join("|");
  useEffect(()=>{
    if(list||!initialized)return;
    const token=++camera.current;
    const raf=requestAnimationFrame(()=>{
      if(token!==camera.current)return;
      if(currentNode){const position=positions.get(currentNode);if(position)void flow.setCenter(position.x+85,position.y+92,{zoom:ref.current && ref.current.clientWidth < 430 ? 0.8 : 1.02,duration:reduced?0:650,ease:t=>1-Math.pow(1-t,3)});}
      else if(ref.current)void flow.setViewport(systemViewport(bounds,ref.current.clientWidth,ref.current.clientHeight),{duration:reduced?0:450});
    });
    return ()=>cancelAnimationFrame(raf);
  },[currentNode,signature,list,reduced,flow,positions,bounds,initialized]);
  function inspectNode(node:GraphNode){onSelect(node.evidence_ids,linkedEvent(node,events,selectedEvent)?.event_id);onInspect({kind:"node",id:node.id});}
  function reset(){setAllContext(true);onInspect(null);}
  const prettyDate=dateDisplay(asOf);
  return <div ref={ref} className={"graph-container universe-canvas "+(list?"is-list":"")} data-initialized={String(initialized)} data-motion={reduced?"reduced":"standard"} data-selected-planet={currentNode??""}>
    <div className="universe-nebula" aria-hidden="true"/><div className="universe-stars" aria-hidden="true"/>
    <div className="graph-toolbar universe-toolbar"><div className="graph-status"><span className="status-dot"/>{nodes.length} planets · {edges.length} connections</div><div className="planet-search"><Search size={14}/><input aria-label="Search planets" placeholder="Find a planet…" value={query} onChange={e=>setQuery(e.target.value)} onKeyDown={e=>{if(e.key==='Escape')setQuery('');if(e.key==='Enter'){const match=original.find(n=>(n.id+' '+display(n.label)+' '+n.type).toLowerCase().includes(query.toLowerCase()));if(query&&match){inspectNode(match);setQuery('')}}}}/>{query&&<div className="planet-search-results">{original.filter(n=>(n.id+' '+display(n.label)+' '+n.type).toLowerCase().includes(query.toLowerCase())).map(n=><button key={n.id} onClick={()=>{inspectNode(n);setQuery('')}}><span>{n.type}</span><strong>{display(n.label)}</strong><code>{n.id}</code></button>)}{!original.some(n=>(n.id+' '+display(n.label)+' '+n.type).toLowerCase().includes(query.toLowerCase()))&&<p>No planet matches this search.</p>}</div>}</div><button className="button small-button universe-ask" data-copilot-trigger onClick={onAsk}><Sparkles size={15}/>Ask Copilot</button></div>
    <details className="universe-view-options"><summary aria-label="Universe view options" title="Universe view options"><Ellipsis size={18}/></summary><div><button onClick={()=>{setList(v=>!v)}}><List size={14}/>{list?'View universe':'Connections'}</button><button onClick={reset}><Maximize2 size={14}/>Show all context</button></div></details>
    {(selection||!allContext)&&<button className="graph-reset button secondary small-button" onClick={reset}><ArrowLeft size={14}/>All context</button>}
    {list?<div className="relation-list"><p className="small muted">Entities and connections as of {prettyDate}. Select an item to inspect details and sources.</p>{nodes.map(n=><button key={n.id} className="relation-row" onClick={()=>inspectNode(n.data.original)}><i className={"planet-swatch planet-"+n.data.original.type.toLowerCase()}/><span className="badge blue">{n.data.original.type}</span><strong>{display(n.data.label)}</strong></button>)}{edges.map(e=><button key={e.id} className="relation-row" onClick={()=>{onSelect(e.data!.original.evidence_ids);onInspect({kind:"edge",id:e.id});}}><span>{original.find(n=>n.id===e.source)?.label} → {original.find(n=>n.id===e.target)?.label}</span><span className="mono small">{e.data!.original.type.replaceAll("_"," ")}</span></button>)}</div>:
    <ReactFlow<PlanetNode,OrbitEdge> nodes={nodes} edges={edges} nodeTypes={nodeTypes} edgeTypes={edgeTypes} fitView fitViewOptions={{padding:.22,maxZoom:.85}} minZoom={.18} maxZoom={1.6} nodesConnectable={false} nodesDraggable={false} edgesReconnectable={false} deleteKeyCode={null} onNodeClick={(_,node)=>inspectNode(node.data.original)} onEdgeClick={(_,edge)=>{onSelect(edge.data!.original.evidence_ids);onInspect({kind:"edge",id:edge.id});}} onEdgeMouseEnter={(_,edge)=>setHoverEdge(edge.id)} onEdgeMouseLeave={()=>setHoverEdge(null)} onPaneClick={()=>{if(selection)reset();}} onKeyDown={event=>{
      if(event.key==="Enter"){const element=(event.target as HTMLElement).closest<HTMLElement>("[data-id]");const node=original.find(n=>n.id===element?.dataset.id);if(node){event.preventDefault();inspectNode(node);}else{const edge=originalEdges.find(e=>e.id===element?.dataset.id);if(edge){event.preventDefault();onSelect(edge.evidence_ids);onInspect({kind:"edge",id:edge.id});}}}
    }} ariaLabelConfig={{"node.a11yDescription.default":"Context planet. Press Enter to focus and inspect connections and sources.","controls.zoomIn.ariaLabel":"Zoom in","controls.zoomOut.ariaLabel":"Zoom out","controls.fitView.ariaLabel":"Fit view"}}><ViewportPortal><div className="solar-systems" aria-hidden="true">{systems.map(system=><div key={system.id} className={'solar-system '+(system.date?'dated-system':'shared-system')} data-system-date={system.date??'shared'} style={{left:system.x,top:system.y,width:system.size,height:system.size}}><div className="solar-system-heading"><span>{system.date?'Solar system':'Shared context'}</span><strong>{system.date?dateDisplay(system.date):'As of '+prettyDate}</strong><small>{system.nodeIds.length} planets{system.date?' · Same source date':' · Persistent source entities'}</small></div><div className="solar-system-orbit"/><div className="solar-system-inner-orbit"/></div>)}</div></ViewportPortal></ReactFlow>}
    {!list&&<><div className="universe-caption"><Orbit size={14}/><span>{currentNode?"Selected planet orbit":"Solar systems grouped by source date"}</span></div><div className="graph-controls"><button aria-label="Zoom in" onClick={()=>void flow.zoomIn({duration:reduced?0:250})}><ZoomIn size={18}/></button><button aria-label="Zoom out" onClick={()=>void flow.zoomOut({duration:reduced?0:250})}><ZoomOut size={18}/></button><button aria-label="Fit universe" onClick={()=>{if(currentNode)void flow.fitView({padding:.22,duration:reduced?0:450,maxZoom:1});else if(ref.current)void flow.setViewport(systemViewport(bounds,ref.current.clientWidth,ref.current.clientHeight),{duration:reduced?0:450});}}><Maximize2 size={17}/></button><button aria-label="Focus selected event" disabled={!selectedEvent||!eventNode(selectedEvent,original)} onClick={()=>{const event=selectedEvent?eventNode(selectedEvent,original):null;if(event)inspectNode(event);}}><LocateFixed size={18}/></button></div></>}
    <div className="graph-legend universe-legend">{["Deal","Account","Person","Event","Interaction","Evidence"].map(type=><span key={type}><i className={"planet-swatch planet-"+type.toLowerCase()}/>{type}</span>)}<span className="legend-edge">— Verified</span><span className="legend-edge">┄ Not verified</span></div>
  </div>;
}
export default function ContextGraph(props:Props){return <ReactFlowProvider><Canvas {...props}/></ReactFlowProvider>;}
