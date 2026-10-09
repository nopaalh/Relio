import Workspace from "@/components/workspace";
export default async function Detail({params}:{params:Promise<{dealId:string}>}){const {dealId}=await params;return <Workspace dealId={dealId}/>;}
