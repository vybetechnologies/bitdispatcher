import { useState } from 'react'
import { api, errText } from '../api'
import type { Person } from '../requests/types'
type Choice={id:string;name:string;dob?:string;person:Person;incidents?:{incident:string;role:string}[]}
export default function RecordPersonPicker({value,onUse}:{value:Person;onUse:(person:Person)=>void}){
 const [query,setQuery]=useState(''),[items,setItems]=useState<Choice[]>([]),[error,setError]=useState(''),[busy,setBusy]=useState(false)
 const search=async(q=query)=>{setBusy(true);setError('');try{const r=await api.post<{items:Choice[];providerMessage?:string}>('/persons/records/search',{query:q});setItems(r.items);if(r.providerMessage)setError(r.providerMessage);else if(!r.items.length)setError('No matching person records')}catch(e){setError(errText(e))}finally{setBusy(false)}}
 return <section className="space-y-2 rounded border border-amber-800/50 bg-amber-950/10 p-3 text-xs" aria-label="Connected person record">
  <div className="flex flex-wrap items-center gap-2"><b className="text-amber-200">{value.recordId?`Connected record · ${value.recordId}`:'Use an existing person record'}</b>{value.recordId&&<button className="btn h-7" type="button" disabled={busy} onClick={()=>void search(value.recordId!)}>Load current linked details</button>}</div>
  <div className="flex gap-2"><input className="input h-8" aria-label="Find person record" placeholder="Name, alias or record ID" value={query} onChange={e=>setQuery(e.target.value)}/><button className="btn h-8" type="button" disabled={busy||query.trim().length<2} onClick={()=>void search()}>Find record</button></div>
  {error&&<p role="status" className="text-amber-200">{error}</p>}
  {items.map(r=><div key={r.id} className="flex flex-wrap items-start gap-2 rounded bg-slate-950/60 p-2"><div className="min-w-0 flex-1"><b>{r.name}</b> · {r.dob||'DOB not recorded'}<div className="text-slate-400">{r.id}{r.incidents?.length?` · Related incidents: ${r.incidents.map(i=>i.incident).join(', ')}`:''}</div></div><button className="btn h-8" type="button" onClick={()=>{onUse({...r.person,recordId:r.id,id:value.id,role:value.role,roles:value.roles,locKind:value.locKind,building:value.building,floor:value.floor,unit:value.unit,clothing:value.clothing,safety:value.safety,safetyNote:value.safetyNote});setItems([])}}>Use this person</button></div>)}
  <p className="text-slate-500">Selecting connects this incident entry to the person record. Incident roles, location, clothing and incident safety concerns stay with this call.</p>
 </section>
}
