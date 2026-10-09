import type { GraphNode, GraphEdge, TimelineEvent } from './contracts';
export type GraphSelection={kind:'node'|'edge';id:string};
export const nodeEventIds=(node:GraphNode):string[]=>node.event_ids??(node.event_id?[node.event_id]:[]);
export function eventNode(eventId:string,nodes:GraphNode[]){
  return nodes.find(n=>n.type==='Event'&&(n.entity_id===eventId||nodeEventIds(n).includes(eventId)||n.id===eventId));
}
export function linkedEvent(node:GraphNode,events:TimelineEvent[],preferred?:string|null){
  const linked=events.filter(e=>nodeEventIds(node).includes(e.event_id));
  return linked.find(e=>e.event_id===preferred)??linked.at(-1);
}
export const entityDescriptions:Record<string,string>={
Deal:'The sales opportunity under review. Stage, value, and assessments depend on the active date and available evidence.',
Account:'The organization providing relationship context. Connections to deals and events follow the available sources.',
Person:'A participant reference from a source. Identity, historical employment and decision authority require their own evidence.',
Event:'A dated event projected from a source: an interaction, request, or decision related to this deal.',
Interaction:'The original communication record, such as an email or meeting note. Source text is distinct from its interpretation.',
Evidence:'A supporting record you can inspect. A precedent from another case does not approve this deal.',
ActionOccurrence:'A sourced historical action. Requested, approved, applied, and completed are distinct states.',
Decision:'A recorded commercial decision. Approval applies only to the object, value, people, and time supported by the source.',
Contract:'Sourced contractual or billing terms. A contract does not automatically prove an action was executed.',
UsageAggregate:'Sourced usage totals. Usage from a comparison customer does not become usage by this prospect.'};
export const relationDescriptions:Record<string,string>={
FOR_ACCOUNT:'This deal is recorded for this account.',
HAS_EVENT:'A source-supported event relationship. Account context does not by itself establish direct deal attribution.',
EXTRACTED_FROM:'This event is projected from this interaction record.',
SUPPORTED_BY:'This entity or claim has this supporting source record.',
SENT_OR_RECORDED:'This person or address sent or recorded the interaction in the source.',
SIMILAR_PRECEDENT:'An inferred comparison precedent. Similarity does not establish approval, consent, or causal uplift.',
RESULTED_IN:'The source records a related outcome; the connection does not establish causality.',
WORKS_AT:'Employment follows the validity period in the historical source. Current roles are not backdated.'};
export function adjacent(id:string,nodes:GraphNode[],edges:GraphEdge[]){
  const index=new Map(nodes.map(n=>[n.id,n]));
  return edges.filter(e=>e.source===id||e.target===id).flatMap(edge=>{
    const neighbor=index.get(edge.source===id?edge.target:edge.source);
    return neighbor?[{edge,neighbor,direction:edge.source===id?"outgoing" as const:"incoming" as const}]:[];
  });
}
export function neighborhood(id:string,nodes:GraphNode[],edges:GraphEdge[]):Set<string>{
  return new Set([id,...adjacent(id,nodes,edges).map(r=>r.neighbor.id)]);
}
export type SolarSystem={id:string;date:string|null;x:number;y:number;size:number;nodeIds:string[]};
// Calendar groups are presentation boundaries, never additional graph entities or relations.
export function solarSystemLayout(nodes:GraphNode[],events:TimelineEvent[]){
  const eventDates=new Map(events.map(event=>[event.event_id,event.date]));
  const groups=new Map<string,GraphNode[]>();
  for(const node of nodes){const dates=[...new Set(nodeEventIds(node).flatMap(id=>eventDates.has(id)?[eventDates.get(id)!]:[]))];const key=dates.length===1?dates[0]:'shared';groups.set(key,[...(groups.get(key)??[]),node]);}
  const keys=[...groups.keys()].sort((a,b)=>a==='shared'?-1:b==='shared'?1:a.localeCompare(b));
  const size=720+Math.max(0,Math.ceil((Math.max(1,...[...groups.values()].map(g=>g.length))-1)/6)-1)*400;
  const systems:SolarSystem[]=[],positions=new Map<string,{x:number;y:number}>();
  keys.forEach((key,index)=>{
    const members=groups.get(key)!.slice().sort((a,b)=>a.id.localeCompare(b.id));
    const center=members.find(n=>n.type==='Event')??members.find(n=>n.type==='Deal')??members[0];
    const others=members.filter(n=>n.id!==center.id);
    const x=index%2*(size+100),y=Math.floor(index/2)*(size+100),cx=x+size/2-85,cy=y+size/2-100;
    systems.push({id:key,date:key==='shared'?null:key,x,y,size,nodeIds:members.map(n=>n.id)});
    positions.set(center.id,{x:cx,y:cy});
    others.forEach((node,i)=>{const ring=Math.floor(i/6),count=Math.min(6,others.length-ring*6),angle=-Math.PI/6+i%6*2*Math.PI/count,radius=245+ring*200;positions.set(node.id,{x:cx+Math.cos(angle)*radius,y:cy+Math.sin(angle)*radius*.8});});
  });
  return {systems,positions};
}
// Stable, bounded layout computed only in the frontend. No force simulation or API writes.
export function orbitalLayout(nodes:GraphNode[],centerId?:string|null){
  const center=nodes.find(n=>n.id===centerId)??nodes.find(n=>n.type==="Deal")??nodes[0];
  if(!center)return new Map<string,{x:number;y:number}>();
  const positions=new Map([[center.id,{x:0,y:0}]]);
  const others=nodes.filter(n=>n.id!==center.id).sort((a,b)=>a.type.localeCompare(b.type)||a.id.localeCompare(b.id));
  others.forEach((node,i)=>{
    const ring=Math.floor(i/8),count=Math.min(8,others.length-ring*8),slot=i%8;
    const angle=(others.length===2?-Math.PI/4:-Math.PI/2)+slot*2*Math.PI/count+ring*.3;
    const radius=others.length<=3?205:270+ring*225;
    positions.set(node.id,{x:Math.round(Math.cos(angle)*radius),y:Math.round(Math.sin(angle)*radius)});
  });
  return positions;
}
