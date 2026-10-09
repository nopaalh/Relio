import type { Metadata } from "next";
import "@fontsource/inter/latin-400.css";
import "@fontsource/inter/latin-500.css";
import "@fontsource/inter/latin-600.css";
import "@fontsource/manrope/latin-600.css";
import "@fontsource/manrope/latin-700.css";
import "@fontsource/jetbrains-mono/latin-400.css";
import "./globals.css";
import "./universe.css";
import "./workspace-vp.css";
import { ChatProvider } from "@/components/chat-context";
export const metadata:Metadata={title:"Relio — Your deal universe. A clearer mind.",description:"Sales context workspace: explore connections, inspect evidence, and compare approaches.",icons:{icon:"/brand/favicon-32.png"}};
export default function Layout({children}:{children:React.ReactNode}) {return <html lang="en"><body><ChatProvider>{children}</ChatProvider></body></html>;}
