import {useEffect,useRef,useState} from 'react'
import type {ReactNode} from 'react'
import type {Person} from '../requests/types'
import {api,errText} from '../api'
import {RecordDetailsDisplay} from '../records/RecordDetails'
export type Associate={record:string;name:string;dob?:string;type:string;notes?:string;incoming?:boolean}
export type Profile={record:string;name?:string;dob?:string;person?:Person;associates?:Associate[];flags?:{id:string;title:string;warning:string;instruction:string}[];seen:string;safetyConcern?:boolean}
export default function PersonProfile({initial,purpose,onBack,onLoaded,renderPerson}:{initial:string;purpose:string;onBack:()=>void;onLoaded:(p:Profile)=>void;renderPerson:(p:Person)=>ReactNode}){
 const [trail,setTrail]=useState([{record:initial,name:'Person profile'}]),[profile,setProfile]=useState<Profile|null>(null),[error,setError]=useState(''),[busy,setBusy]=useState(false)
 const request=useRef(0),loaded=useRef(onLoaded);loaded.current=onLoaded
 const current=trail[trail.length-1].record
 useEffect(()=>{const id=++request.current;setBusy(true);setError('');setProfile(null);api.post<Profile>('/mdt/records/person',{record:current,purpose}).then(p=>{if(id!==request.current)return;setProfile(p);setTrail(t=>t.map(v=>v.record===p.record?{...v,name:p.name||v.name}:v));loaded.current(p)}).catch(e=>{if(id===request.current)setError(errText(e))}).finally(()=>{if(id===request.current)setBusy(false)});return()=>{request.current++}},[current,purpose])
 const open=(record:string,name:string)=>{setTrail(t=>{const found=t.findIndex(p=>p.record===record);return found>=0?t.slice(0,found+1):[...t.slice(-63),{record,name}]})}
 return <section className="space-y-3" aria-label="Person profile"><div className="flex flex-wrap gap-2"><button className="btn" onClick={onBack}>Back to search results</button>{trail.length>1&&<button className="btn" onClick={()=>setTrail(t=>t.slice(0,-1))}>Previous person</button>}</div><nav className="flex flex-wrap gap-1 text-xs" aria-label="Profile path">{trail.map((p,i)=><button className="btn" key={p.record} disabled={busy||i===trail.length-1} onClick={()=>setTrail(t=>t.slice(0,i+1))}>{i+1}. {i===trail.length-1?profile?.name||p.name:p.name}</button>)}</nav>
  {busy&&<p role="status">Loading local person profile…</p>}{error&&<p role="alert" className="text-rose-300">{error}</p>}
  {profile&&<><h3 className="text-lg font-semibold">{profile.name||'Unnamed person'}</h3><p className="text-xs text-slate-400">{profile.record} · DOB {profile.dob||'not recorded'}</p><RecordDetailsDisplay person={profile.person}/>{profile.person?.agencyEmployee&&<p className="text-teal-200">LAS employee</p>}{profile.flags?.map(f=><section key={f.id} className="rounded border border-rose-700 p-2 text-rose-200"><b>Officer safety · {f.title}</b><p>{f.warning}</p><p>{f.instruction}</p></section>)}{profile.person&&renderPerson(profile.person)}
   <section className="space-y-2" aria-label="Associates"><h4 className="font-semibold">Relationships & associates</h4><p className="text-xs text-slate-400">Relationships describe each linked person relative to this person. Each profile’s cautions apply to that person.</p>{!profile.associates?.length&&<p className="text-sm text-slate-400">No linked associates available.</p>}{profile.associates?.map(a=><article className="rounded border border-slate-600 p-3 text-sm" key={a.record+':'+a.type}><button className="btn text-amber-200" onClick={()=>open(a.record,a.name)}>{a.type} · {a.name}</button><p className="mt-1 text-xs text-slate-400">DOB {a.dob||'not recorded'} · {a.record}</p>{a.notes&&<p>{a.notes}</p>}</article>)}</section>
  </>}
 </section>
}
