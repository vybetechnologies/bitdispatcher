import { useEffect, useState } from 'react'
import { Download, Monitor } from 'lucide-react'
import { api } from './api'
import { isWindowsMdt } from './desktop'
export default function DesktopDownload({compact=false}:{compact?:boolean}){
 const [release,setRelease]=useState<{Available:boolean;Version?:string;Message?:string}|null>(null)
 useEffect(()=>{
  if(isWindowsMdt())return
  let active=true
  const check=()=>void api.get<{Available:boolean;Version?:string;Message?:string}>('/mdt/windows-release').then(value=>{if(active)setRelease(value)}).catch(()=>{if(active)setRelease({Available:false,Message:'Download temporarily unavailable'})})
  check();const timer=setInterval(check,60000);return()=>{active=false;clearInterval(timer)}
 },[])
 if(isWindowsMdt())return null
 return <section aria-label="Windows MDT application" className={compact?'space-y-2':'rounded-lg border border-slate-700 bg-slate-900/60 p-3'}>
  {!compact&&<p className="mb-2 flex gap-2 text-sm font-semibold text-slate-200"><Monitor size={16}/> LAS MDT for Windows</p>}
  {release?.Available?<a className="btn inline-flex min-h-9 items-center gap-2 text-xs" href="./api/mdt/windows-download" title={'Download Windows MDT '+(release.Version||'')} aria-label={'Download Windows MDT '+(release.Version||'')}><Download size={14}/> Download Windows MDT{release.Version?' '+release.Version:''}</a>:<p className="text-xs text-slate-400">{release?.Message||'Checking Windows download…'}</p>}
  {!compact&&<p className="mt-2 text-xs text-slate-400">Windows 10/11 · own-unit MDT · offline work and automatic sync · updates inside the app. Sign in online once to prepare this device.</p>}
 </section>
}
