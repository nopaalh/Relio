"use client";
import { createContext, useContext, useState } from "react";
import type { CopilotResponse } from "@/lib/contracts";
export type ChatEntity={id:string;type:string;label:string;evidenceIds:string[]};
export type ChatMessage={id:string;question:string;date:string;dealId:string|null;entity?:ChatEntity;response?:CopilotResponse;error?:string;unavailable?:boolean};
const ChatContext=createContext<{messages:ChatMessage[];setMessages:React.Dispatch<React.SetStateAction<ChatMessage[]>>}|null>(null);
export function ChatProvider({children}:{children:React.ReactNode}){const [messages,setMessages]=useState<ChatMessage[]>([]);return <ChatContext.Provider value={{messages,setMessages}}>{children}</ChatContext.Provider>;}
export function useChat(){const state=useContext(ChatContext);if(!state)throw new Error("Chat provider missing");return state;}
