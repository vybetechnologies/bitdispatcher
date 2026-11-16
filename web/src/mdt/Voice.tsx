import { useEffect, useRef, useState } from 'react'
import { Volume2, VolumeX, RotateCcw } from 'lucide-react'
import type { MDTView } from './Mdt'
import type { CommView } from '../types'
import { statusMeta } from '../util'
import type { WeatherFeature } from '../WeatherAlerts'

export type Announcement = { id: string; text: string; priority: number }
export type LookupCaution = {id:number;text:string}
const words = (s?: string) => (s || '').replace(/\s+/g,' ').trim().slice(0,600)
/** Input is only the assigned unit's already-authorized terminal/inbox data. */
export function voiceSnapshot(m: MDTView, messages: CommView[], weather: WeatherFeature[] = [],lookupCaution?:LookupCaution|null): Announcement[] {
  const out: Announcement[]=[]
  const add=(id:string,text:string,priority=2)=>out.push({id,text:words(text),priority})
  const i=m.incident
  add('assignment',i ? `Assigned call ${i.number}. ${m.typeName || i.type}. Priority ${i.priority}. ${words(i.location?.address)}.` : 'No call assigned.')
  add('status',`Unit ${m.unit.id}. Status ${statusMeta(m.unit.status).label}.`,1)
  for(const call of m.pending||[])add('pending:'+call.number,`New call for service. ${call.type}. Priority ${call.priority}. ${words(call.location?.address)}. Call ${call.number}.`,call.priority<=2?1:2)
  if(lookupCaution)add('lookup-caution:'+lookupCaution.id,lookupCaution.text,-1)
  if(i) {
    add('call-status',`Call ${i.number}. ${i.status}. Priority ${i.priority}.`)
    if(i.newLocation) add('location-change',`Dispatch changed the call location. ${words(i.newLocation.location.address)}`,0)
    for(const f of i.flags || []) add('flag:'+f.id,`Safety flag. ${words(f.title)}. ${words(f.severity)}`,0)
    if(m.alarm) add('alarm',`Call alert. ${words(m.alarm)}`,0)
    for(const p of m.premises || []) if(p.hazards?.length) add('premise:'+p.id,`Site safety warning. ${words(p.name)}. ${words(p.hazards.join('. '))}`,0)
    if(m.papers?.count) add('papers',`Safety warning. ${m.papers.count} active papers. ${words(m.papers.caution)}`,0)
    for(const c of m.cautions || []) add('caution:'+c.id,`Safety warning. ${words(c.name)}. ${words(c.note)}`,0)
    for(const n of (i.timeline || []).filter(n=>n.priority).slice(-5)) add(`note:${i.number}:${n.seq}`,`Priority dispatch note. ${words(n.text)}`,0)
  }
  if(m.emergency) add('emergency',m.emergency.ackedBy ? 'Dispatch acknowledged your emergency.' : 'Emergency sent. Waiting for dispatch acknowledgement.',-2)
  for(const message of (m.messages || []).filter(x=>x.from!==m.unit.id).slice(-12)) add('message:'+message.id,`${message.urgent?'Urgent message':'Message'} from ${message.fromUnit ? 'unit ' : ''}${words(message.from)}. ${words(message.text)}`,message.urgent?0:2)
  for(const b of m.bolos || []) add('bolo:'+b.id,`BOLO. ${words(b.text)}. ${b.plate ? 'Plate '+words(b.plate) : ''}`,1)
  for(const c of messages.filter(c=>(c.state==='active'||c.state==='updated') && (!c.expires || Date.parse(c.expires)>Date.now()))) {
    if(!c.mine) add('comm:'+c.id,`${c.kind==='safety'?'Safety warning':c.kind==='bolo'?'BOLO':'Message'}. ${words(c.subject)}. ${words(c.body)}`,c.kind==='safety'||c.priority==='urgent'?0:c.priority==='high'||c.kind==='bolo'?1:2)
  }
  for(const f of weather) {
    const p=f.properties
    if(Date.parse(p.expires)<=Date.now() || (p.ends && Date.parse(p.ends)<=Date.now()))continue
    add('weather:'+f.id,`National Weather Service. ${words(p.event)} for ${words(p.areaDesc)}. ${words(p.headline)}. ${words(p.instruction || p.description)}`,/warning/i.test(p.event)||p.severity==='Extreme'||p.severity==='Severe'?0:1)
  }
  return out.sort((a,b)=>a.priority-b.priority)
}
export function changedAnnouncements(before: Map<string,string>, next: Announcement[]) {
  return next.filter(n=>before.get(n.id)!==n.text)
}

export default function MdtVoice({ m, messages, weather = [],lookupCaution, connected }: {m:MDTView|null;messages:CommView[];weather?:WeatherFeature[];lookupCaution?:LookupCaution|null;connected:boolean}) {
  const [enabled,setEnabled]=useState(false)
  const [voices,setVoices]=useState<SpeechSynthesisVoice[]>([])
  const [voice,setVoice]=useState('')
  const [rate,setRate]=useState(1)
  const [volume,setVolume]=useState(1)
  const [settings,setSettings]=useState(false)
  const [error,setError]=useState('')
  const [waitingForTap,setWaitingForTap]=useState(false)
  const attemptedDefault=useRef(false)
  const seen=useRef(new Map<string,string>())
  const pending=useRef<Announcement[]>([])
  const speaking=useRef(false)
  const currentSpoken=useRef<Announcement|null>(null)
  const last=useRef('')
  const generation=useRef(0)
  const enableToken=useRef(0)
  const release=useRef<(()=>void)|null>(null)
  const supported=typeof speechSynthesis!=='undefined'
  const prefs=useRef({voice,rate,volume});prefs.current={voice,rate,volume}
  const locals=useRef(voices);locals.current=voices
  const active=useRef(enabled);active.current=enabled
  const stop=()=>{generation.current++;pending.current=[];speaking.current=false;currentSpoken.current=null;if(supported)speechSynthesis.cancel()}
  const off=()=>{enableToken.current++;active.current=false;setEnabled(false);setWaitingForTap(false);stop();release.current?.();release.current=null}
  useEffect(()=>{
    if(!supported)return
    const load=()=>setVoices(speechSynthesis.getVoices().filter(v=>v.localService && /^en(?:-|$)/i.test(v.lang)))
    load();speechSynthesis.addEventListener('voiceschanged',load)
    return()=>speechSynthesis.removeEventListener('voiceschanged',load)
  },[supported])
  const pump=()=>{
    if(!active.current||speaking.current||!pending.current.length)return
    const a=pending.current.shift()!
    const selected=locals.current.find(v=>v.voiceURI===prefs.current.voice)||locals.current[0]
    if(!selected){off();setError('No local English voice is available. Install a device voice.');return}
    const token=generation.current
    const u=new SpeechSynthesisUtterance(a.text)
    u.voice=selected;u.lang=selected.lang;u.rate=prefs.current.rate;u.volume=prefs.current.volume
    speaking.current=true;currentSpoken.current=a;last.current=a.text
    u.onend=()=>{if(token!==generation.current)return;speaking.current=false;currentSpoken.current=null;pump()}
    u.onerror=(e)=>{if(token!==generation.current)return;off();if(e.error==='not-allowed')setWaitingForTap(true);setError(e.error==='not-allowed'?'Browser requires a tap. Select Enable audio to start speaking.':'Voice stopped. Check device audio, then enable voice again.')}
    speechSynthesis.speak(u)
  }
  const enqueue=(items:Announcement[])=>{
    if(speaking.current && items.some(i=>i.priority<(currentSpoken.current?.priority ?? 2))) {
      const interrupted=currentSpoken.current
      const rest=pending.current
      stop()
      pending.current=[...(interrupted && interrupted.id!=='ready' ? [interrupted] : []),...rest]
    }
    const merged=new Map(pending.current.map(i=>[i.id,i]))
    items.forEach(i=>merged.set(i.id,i))
    pending.current=[...merged.values()].sort((a,b)=>a.priority-b.priority).slice(0,12)
    pump()
  }
  const enable=async()=>{
    setError('')
    setWaitingForTap(false)
    if(!supported||!voices.length){setError('A local English device voice is required.');return}
    const token=++enableToken.current
    const start=()=>{seen.current.clear();active.current=true;setEnabled(true);enqueue([{id:'ready',text:'LAS terminal voice enabled.',priority:0}])}
    if(navigator.locks) {
      // Keep one speaking MDT per browser origin; no double announcements in tabs.
      navigator.locks.request('las-mdt-voice',{ifAvailable:true},async lock=>{
        if(token!==enableToken.current)return
        if(!lock){setError('Voice is already enabled in another LAS tab. Mute that tab first.');return}
        await new Promise<void>(resolve=>{release.current=resolve;start()})
      }).catch(()=>setError('Unable to enable voice in this browser.'))
    } else start()
  }
  useEffect(()=>{
    if(!supported || !voices.length || attemptedDefault.current)return
    attemptedDefault.current=true
    void enable()
    // Browsers may require interaction; the explicit Voice button remains available.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  },[supported,voices])
  useEffect(()=>{
    if(!enabled||!m)return
    if(!connected){
      // A completed person search is fresh authenticated data even if the
      // separate live call stream is reconnecting. Still speak its caution.
      const key=lookupCaution?'lookup-caution:'+lookupCaution.id:''
      pending.current=pending.current.filter(n=>n.id===key)
      if(currentSpoken.current?.id!==key)stop()
      const text=words(lookupCaution?.text)
      if(lookupCaution&&seen.current.get(key)!==text){seen.current.set(key,text);enqueue([{id:key,text,priority:-1}])}
      return
    }
    const next=voiceSnapshot(m,messages,weather,lookupCaution)
    // Seed the current pending queue on entry; announce new calls rather than
    // reading an old backlog each time the officer opens the terminal.
    const firstSnapshot=seen.current.size===0
    const changed=changedAnnouncements(seen.current,next).filter(n=>!firstSnapshot||!n.id.startsWith('pending:'))
    const current=currentSpoken.current
    if((current && seen.current.has(current.id) && !next.some(n=>n.id===current.id && n.text===current.text)) ||
      (seen.current.has('assignment') && changed.some(n=>n.id==='assignment'))) {
      const rest=pending.current
      stop()
      pending.current=rest
    }
    // Remember delivered snapshots across temporary feed outages, so recovery
    // does not replay unchanged messages/weather. Bound this session history.
    for(const n of next)seen.current.set(n.id,n.text)
    while(seen.current.size>512)seen.current.delete(seen.current.keys().next().value!)
    // Drop withdrawn/superseded items, including cleared assignments and BOLOs.
    const live=new Map(next.map(n=>[n.id,n.text]))
    pending.current=pending.current.filter(n=>live.get(n.id)===n.text)
    enqueue(changed)
    // Speech callbacks read current settings through refs.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  },[enabled,m,messages,weather,lookupCaution,connected])
  useEffect(()=>{
    const cancel=()=>off()
    window.addEventListener('bd:locked',cancel);window.addEventListener('bd:signedout',cancel)
    return()=>{window.removeEventListener('bd:locked',cancel);window.removeEventListener('bd:signedout',cancel);attemptedDefault.current=false;off()}
    // eslint-disable-next-line react-hooks/exhaustive-deps
  },[])
  return <div className="flex flex-wrap items-center gap-2">
    <button className={`btn h-11 ${enabled||waitingForTap?'border-amber-400 text-amber-200':''}`} aria-pressed={enabled||waitingForTap} disabled={!supported} onClick={()=>enabled?off():void enable()}>{enabled||waitingForTap?<Volume2 size={17}/>:<VolumeX size={17}/>} {waitingForTap?'Voice on · Enable audio':`Voice ${enabled?'on':'off'}`}</button>
    {enabled&&<><button className="btn h-11" aria-label="Repeat last voice announcement" onClick={()=>{stop();enqueue([{id:'repeat',text:last.current||'No previous announcement.',priority:1}])}}><RotateCcw size={16}/> Repeat</button><button className="btn h-11" onClick={stop}>Stop speaking</button></>}
    <button className="btn h-9" aria-expanded={settings} onClick={()=>setSettings(v=>!v)}>Voice settings</button>
    {(!supported||error)&&<span role="status" className="text-xs text-amber-200">{error||'Speech is unavailable in this browser.'}</span>}
    {enabled&&!connected&&<span role="status" className="text-xs text-amber-200">Live call announcements paused · reconnecting</span>}
    {settings&&<div className="panel flex w-full flex-wrap items-center gap-3 rounded p-3 text-sm">
      <label>Device voice <select className="input" value={voice} onChange={e=>setVoice(e.target.value)}><option value="">Default local English voice</option>{voices.map(v=><option key={v.voiceURI} value={v.voiceURI}>{v.name}</option>)}</select></label>
      <label>Speed <input aria-label="Voice speed" type="range" min="0.75" max="1.4" step="0.05" value={rate} onChange={e=>setRate(Number(e.target.value))}/>{rate.toFixed(2)}×</label>
      <label>Volume <input aria-label="Voice volume" type="range" min="0.1" max="1" step="0.1" value={volume} onChange={e=>setVolume(Number(e.target.value))}/></label>
      <p className="w-full text-xs text-slate-400">Voice starts automatically. If the browser blocks audio, tap Voice on. Uses installed local voices; no microphone. Speech does not acknowledge messages.</p>
    </div>}
  </div>
}
