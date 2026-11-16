import DesktopStatus from './DesktopStatus'
import { useEffect, useRef, useState } from 'react'
import { ChevronDown, Layers3, Check } from 'lucide-react'
import { useLocation, useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { api, ApiError, errText } from './api'
import { useStore } from './store'
import { clearSnapshot, pauseSnapshots, resumeSnapshots } from './offline'
import type { Session } from './types'

export type WorkspaceModule = { id:string; label:string; path:string }
export function activeModule(path:string,role?:string) {
 if(role==='field'&&!path.startsWith('/rms'))return 'mdt'
 if(path.startsWith('/mdt'))return 'mdt'
 if(path.startsWith('/admin'))return 'admin'
 if(path.startsWith('/rms')||path.startsWith('/records'))return 'rms'
 if(path.startsWith('/taker'))return 'taker'
 return 'cad'
}
const TRANSITION='las.module-transition'
export default function ModuleSwitcher() {
 const me=useStore(s=>s.me)
 const location=useLocation(),navigate=useNavigate()
 const [modules,setModules]=useState<WorkspaceModule[]>([])
 const [busy,setBusy]=useState(false)
 const [pending,setPending]=useState(false)
 const menu=useRef<HTMLDetailsElement>(null)
 const refresh=()=>api.get<{modules:WorkspaceModule[]}>('/auth/modules').then(r=>{setModules(r.modules);return r.modules})
 useEffect(()=>{
  let alive=true
  const update=()=>api.get<{modules:WorkspaceModule[]}>('/auth/modules').then(r=>{if(alive)setModules(r.modules)}).catch(()=>{if(alive)setModules([])})
  update();const timer=setInterval(update,60000)
  window.addEventListener('focus',update)
  return()=>{alive=false;clearInterval(timer);window.removeEventListener('focus',update)}
 },[me?.username,me?.duty])
 useEffect(()=>{
  const receive=(e:StorageEvent)=>{
   if(e.key!==TRANSITION||!e.newValue)return
   try{const notice=JSON.parse(e.newValue) as {user:string;stage:string};if(notice.user!==me?.username)return
    if(notice.stage==='start'){pauseSnapshots();setBusy(true)}
    if(notice.stage==='cancel'){resumeSnapshots();setBusy(false)}
    if(notice.stage==='done'){pauseSnapshots();setBusy(true);void clearSnapshot().then(()=>window.location.reload())}
   }catch{/* Ignore unrelated storage content. */}
  }
  const outside=(e:PointerEvent)=>{if(menu.current&&!menu.current.contains(e.target as Node))menu.current.open=false}
  window.addEventListener('storage',receive);document.addEventListener('pointerdown',outside)
  return()=>{window.removeEventListener('storage',receive);document.removeEventListener('pointerdown',outside)}
 },[me?.username])
 const notify=(stage:string)=>localStorage.setItem(TRANSITION,JSON.stringify({user:me?.username,stage,at:Date.now()}))
 const select=async(module:WorkspaceModule)=>{
  if(pending||busy)return
  setPending(true);pauseSnapshots();notify('start')
  try{
   const result=await api.post<{path:string;changed:boolean;session:Session}>('/auth/module',{module:module.id})
   if(menu.current)menu.current.open=false
   if(!result.changed){notify('cancel');resumeSnapshots();navigate(result.path);return}
   setBusy(true);localStorage.setItem('bd.me',JSON.stringify(result.session))
   notify('done');await clearSnapshot();window.location.hash=result.path;window.location.reload()
  }catch(e){
   if(e instanceof ApiError){notify('cancel');resumeSnapshots();toast.error(errText(e));void refresh().catch(()=>{})}
   else{setBusy(true);notify('done');await clearSnapshot();window.location.reload()}
  }finally{setPending(false)}
 }
 const current=modules.find(m=>m.id===activeModule(location.pathname,me?.role))
 if(modules.length<=1&&!busy)return null
 return <>
  {modules.length>1&&<nav className="flex h-10 shrink-0 items-center gap-2 border-b border-slate-800 bg-slate-950 px-3 text-xs" aria-label="Workspace modules">
   <span className="hidden text-[10px] font-semibold uppercase tracking-wider text-slate-500 sm:inline">LAS Operations</span>
   <details ref={menu} className="relative" onKeyDown={e=>{if(e.key==='Escape'&&menu.current){menu.current.open=false;menu.current.querySelector('summary')?.focus()}}}>
    <summary aria-label="Switch module" className="btn flex h-7 cursor-pointer list-none items-center gap-2 border-amber-600/40 px-2.5 text-xs text-amber-200 [&::-webkit-details-marker]:hidden" onClick={()=>{void refresh().catch(()=>{})}}><Layers3 size={14} aria-hidden />{current?.label||'Modules'}<ChevronDown size={12} aria-hidden /></summary>
    <div className="panel absolute left-0 top-full z-40 mt-2 w-60 overflow-hidden rounded-lg p-1 shadow-2xl" aria-label="Available modules">
     {modules.map(module=><button key={module.id} className="flex min-h-10 w-full items-center gap-2 rounded px-3 py-2 text-left text-sm text-slate-200 hover:bg-slate-800 focus-visible:bg-slate-800 disabled:opacity-50" disabled={pending||busy} onClick={()=>select(module)}><span className="flex-1">{module.label}</span>{module.id===current?.id&&<Check size={14} className="text-amber-300" aria-label="Current module" />}</button>)}
     <p className="border-t border-slate-800 px-3 py-2 text-[10px] leading-relaxed text-slate-500">Other LAS tabs refresh when your working duty changes.</p>
    </div>
   </details>
   <span className="ml-auto truncate text-[11px] text-slate-500">{me?.name}</span>
  </nav>}
  {busy&&<div role="status" className="fixed inset-0 z-[100] grid place-items-center bg-slate-950 text-sm text-slate-200">Switching workspace…</div>}
 </>
}

/** Shared shell puts the same switcher above every signed-in screen without covering its controls. */
export function WorkspaceShell({children}:{children:React.ReactNode}) {
 const me=useStore(s=>s.me)
 if(!me)return <>{children}</>
 return <div className="flex h-full min-h-0 flex-col"><ModuleSwitcher /><DesktopStatus/><div className="min-h-0 flex-1">{children}</div></div>
}
