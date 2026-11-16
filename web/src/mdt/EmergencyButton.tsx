import { useEffect, useRef, useState } from 'react'
import { Siren } from 'lucide-react'
import { toast } from 'sonner'
import { api, errText } from '../api'
import { isWindowsMdt, desktopState } from '../desktop'

/** Always visible in the fixed-height MDT emergency dock. Never sends on tap. */
export default function EmergencyButton({unitParam,onSent}:{unitParam:string;onSent:()=>void}){
 const [held,setHeld]=useState(0)
 const timer=useRef<number|undefined>(undefined)
 const busy=useRef(false)
 const stop=()=>{clearInterval(timer.current);timer.current=undefined;setHeld(0)}
 const start=()=>{
  if(timer.current!==undefined||busy.current)return
  const t0=Date.now()
  timer.current=window.setInterval(()=>{
   const f=Math.min(1,(Date.now()-t0)/2000);setHeld(f)
   if(f<1)return
   stop();busy.current=true
   api.post<{queued?:boolean}>('/mdt/emergency'+unitParam).then(result=>{toast.error(result?.queued||isWindowsMdt()&&!desktopState().connected?'EMERGENCY SAVED — NOT SENT. Use radio or phone for help.':'EMERGENCY sent to dispatch');onSent()}).catch(e=>toast.error(errText(e))).finally(()=>{busy.current=false})
  },50)
 }
 useEffect(()=>{
  const blur=()=>stop();window.addEventListener('blur',blur)
  return()=>{clearInterval(timer.current);window.removeEventListener('blur',blur)}
 },[])
 return <button className="relative h-12 w-full shrink-0 touch-none select-none overflow-hidden rounded-lg border border-rose-700 bg-rose-950 text-base font-black tracking-wider text-rose-200"
  onPointerDown={e=>{if(e.button===0)start()}} onPointerUp={stop} onPointerLeave={stop} onPointerCancel={stop}
  onKeyDown={e=>{if(e.key===' '||e.key==='Enter'){e.preventDefault();if(!e.repeat)start()}}} onKeyUp={e=>{if(e.key===' '||e.key==='Enter'){e.preventDefault();stop()}}} onBlur={stop}
  aria-label="Emergency: hold for two seconds"><span className="absolute inset-y-0 left-0 bg-rose-600" style={{width:`${held*100}%`}}/><span className="relative flex items-center justify-center gap-2"><Siren size={18} aria-hidden/> HOLD FOR EMERGENCY</span></button>
}
