import { useEffect, useState } from 'react'
import { CheckCircle2, CloudOff, RefreshCw, TriangleAlert } from 'lucide-react'
import { desktopOutbox, desktopState, discardDesktopCommand, isWindowsMdt, retryDesktopSync, startDesktopSync, recoverLegacyLiveEntries } from './desktop'
import type { DesktopCommand } from './desktop'

export default function DesktopStatus(){
 const [status,setStatus]=useState(desktopState),[open,setOpen]=useState(false),[items,setItems]=useState<DesktopCommand[]>([])
 useEffect(()=>{
  if(!isWindowsMdt())return
  const change=()=>{setStatus({...desktopState()});if(open)void desktopOutbox().then(setItems)}
  window.addEventListener('las:desktop',change);change()
  const stop=startDesktopSync()
  return()=>{stop();window.removeEventListener('las:desktop',change)}
 },[open])
 if(!isWindowsMdt())return null
 return <div className="relative shrink-0 border-b border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-200">
  <div className="flex flex-wrap items-center gap-2" role="status">
   {status.connected?<CheckCircle2 size={16} className="text-emerald-400"/>:<CloudOff size={16} className="text-amber-300"/>}
   <b>{status.connected?'Windows MDT · connected':'OFFLINE · saved data on this device'}</b>
   <button className="btn h-8 text-xs" onClick={()=>{setOpen(!open);void desktopOutbox().then(setItems)}}>{status.pending} pending sync</button>
   <button className="btn h-8 text-xs" onClick={()=>void retryDesktopSync()}><RefreshCw size={13}/> Sync now</button>
   {status.preparing&&<span className="text-xs text-amber-200">Preparing offline records and recorded maps…</span>}
   {!status.connected&&<span className="text-amber-200">Calls, messages and emergency requests cannot reach dispatch until connected. Use radio or phone for urgent help.</span>}
  </div>
  {status.error&&<p className="mt-1 flex gap-2 text-xs text-amber-200"><TriangleAlert size={14}/>{status.error}</p>}
  {open&&<section aria-label="Windows MDT sync outbox" className="absolute inset-x-0 top-full z-50 mx-3 max-h-[45vh] space-y-2 overflow-y-auto rounded-lg border border-slate-700 bg-slate-950 p-3 shadow-xl">
   {items.some(item=>!item.Context)&&<button className="btn text-xs" onClick={()=>{if(confirm('Confirm these older unlabeled entries were ALL recorded during LIVE LAS operations. Training work must not be sent to the live board. Originals will be backed up on this device.'))void recoverLegacyLiveEntries().then(()=>desktopOutbox().then(setItems)).catch(e=>alert(e instanceof Error?e.message:String(e)))}}>Recover older live entries</button>}
   {!items.length&&<p className="text-xs text-slate-400">All saved work has synced.</p>}
   {items.map(item=><article key={item.ID} className="rounded border border-slate-700 p-2 text-xs">
    <div className="flex flex-wrap items-center gap-2"><b>{item.Path}</b><span>{new Date(item.At).toLocaleString()} · {item.Unit}</span><span className="ml-auto">{item.error?'Review needed':'Waiting to sync'}</span></div>
    {(item.Body.text||item.Body.status||item.FileName)&&<p className="mt-1 whitespace-pre-wrap text-slate-300">{String(item.Body.text||item.Body.status||item.FileName)}</p>}
    {item.error&&<p className="mt-1 text-amber-200">{item.error}</p>}
    <div className="mt-2 flex gap-2"><button className="btn h-7 text-xs" onClick={()=>{const blob=new Blob([JSON.stringify(item,null,2)],{type:'application/json'});const url=URL.createObjectURL(blob);const a=document.createElement('a');a.href=url;a.download='LAS-offline-'+item.ID+'.json';a.click();URL.revokeObjectURL(url)}}>Export saved entry</button>
    <button className="btn h-7 text-xs text-rose-300" onClick={()=>{if(confirm('Discard this saved entry from this device? Export it first if it is needed. An action already received by dispatch cannot be recalled.'))void discardDesktopCommand(item.ID).then(()=>desktopOutbox().then(setItems))}}>Discard local entry</button></div>
   </article>)}
  </section>}
 </div>
}
