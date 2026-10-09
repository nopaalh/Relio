"use client";
export default function ErrorPage({reset}:{reset:()=>void}){return <main className="route-loading"><h1>Workspace could not be loaded</h1><p>Try reloading this page.</p><button onClick={reset}>Try again</button></main>;}
