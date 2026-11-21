import { RecordDetailsDisplay, recordDetailsAnnouncement } from '../records/RecordDetails'
import type {RecordCaution} from '../records/RecordDetails'
import { useEffect, useRef, useState } from 'react'
import { Car, Search, ShieldAlert, UserRound } from 'lucide-react'
import { api, errText } from '../api'
import { Modal } from '../ui'
import { cx } from '../util'
import FbiWanted from './FbiWanted'
import type {FbiQuery} from './FbiWanted'
import type { Person } from '../requests/types'

type RegisteredOwner={record:string;name:string;dob?:string;person?:Person;flags?:{id:string;title:string;warning:string;instruction:string}[]}
type Hit = {registeredOwner?:RegisteredOwner; cautions?:RecordCaution[];ownership?:string;ownerName?:string; flags?:{id:string;title:string;warning:string;instruction:string}[];person?:Person;incidents?:{incident:string;role:string}[];source?:string; record:string; incident:string; seen:string; name?:string; dob?:string; sex?:string; description?:string; safetyConcern?:boolean; plate?:string; state?:string; vin?:string; year?:string; make?:string; model?:string; color?:string; body?:string }
type Results = { providerMessage?:string; kind:'person'|'vehicle'; items:Hit[]; more:boolean; source:string; searchedAt:string }
export function personCautionAnnouncement(result:Results):string {
 const extra=result.items.map(h=>{const text=recordDetailsAnnouncement(h.person,h.cautions,h.ownership,h.ownerName);return text?`Possible match ${h.name||h.plate||h.vin||'record'}. ${text}`:''}).filter(Boolean).join(' ')
 if(result.kind!=='person')return [extra,...result.items.filter(h=>h.registeredOwner).map(h=>{const o=h.registeredOwner!;const details=recordDetailsAnnouncement(o.person);const core=(o.flags||[]).map(f=>[f.title,f.warning,f.instruction].filter(Boolean).join('. ')).join('. ');return `Recorded registered owner for vehicle ${h.plate||h.vin||'record'}: ${o.name}. ${details} ${core?`Registered owner safety flag. ${core}`:''}`})].filter(Boolean).join(' ')
 const cautions=result.items.filter(h=>h.safetyConcern||h.person?.safety||h.flags?.length)
 const employees=result.items.filter(h=>h.person?.agencyEmployee)
 const employeeInfo=employees.length?`Agency employee information. ${employees.slice(0,3).map(h=>`Possible match ${h.name||'unnamed person'} is an employee of Lakes Area Security.`).join(' ')}`:''
 if(!cautions.length)return [extra,employeeInfo].filter(Boolean).join(' ')
 const details=cautions.slice(0,3).map(h=>{
  const flags=(h.flags||[]).map(f=>[f.title,f.warning,f.instruction].filter(Boolean).join('. '))
  const n=h.person?.safetyNote
  if(!flags.length&&n)flags.push([n.title,n.reason,n.weapons,n.instruction].filter(Boolean).join('. '))
  return `Possible match ${h.name||'unnamed person'}. ${flags.join('. ')||'Recorded caution. Contact dispatch for details.'}`
 })
 return `Officer safety warning. ${cautions.length} possible person ${cautions.length===1?'match has a caution':'matches have cautions'}. ${details.join('. ')}${cautions.length>3?'. Additional cautions are shown in the search results.':''} ${employeeInfo} ${extra}`.trim()
}

/** Private local-record lookup, kept inside MDT so it never changes the user's duty or assigned unit. */
export default function RecordSearch({onClose,onCaution,onFbiCaution}:{onClose:()=>void;onCaution?:(text:string)=>void;onFbiCaution?:(text:string)=>void}) {
 const [fbiQuery,setFbiQuery]=useState<FbiQuery|null>(null)
 const [kind,setKind]=useState<'person'|'vehicle'>('person')
 const [name,setName]=useState('')
 const [dob,setDob]=useState('')
 const [plate,setPlate]=useState('')
 const [state,setState]=useState('')
 const [vin,setVin]=useState('')
 const [purpose,setPurpose]=useState('assigned-call')
 const [busy,setBusy]=useState(false)
 const [error,setError]=useState('')
 const [result,setResult]=useState<Results|null>(null)
 const request=useRef(0)
 useEffect(()=>()=>{request.current++},[])
 const changeKind=(next:'person'|'vehicle')=>{request.current++;setKind(next);setError('');setResult(null);setFbiQuery(null);onFbiCaution?.('')}
 const search=async(e:React.FormEvent)=>{
  e.preventDefault();if(busy)return;const id=++request.current;setBusy(true);setError('');setResult(null)
  onFbiCaution?.('');setFbiQuery(kind==='person'?{id,name,dob,purpose}:null)
  try{const response=await api.post<Results>('/mdt/records/search',{kind,purpose,...(kind==='person'?{name,dob}:{plate,state,vin})});if(request.current===id){setResult(response);onCaution?.(personCautionAnnouncement(response))}}
  catch(e){if(request.current===id)setError(errText(e))}
  finally{if(request.current===id)setBusy(false)}
 }
 return <Modal open onClose={onClose} title="Persons & vehicles" icon={<Search size={18} className="text-amber-300" />} wide>
  <div className="space-y-4">
   <div className="flex gap-2" role="tablist" aria-label="Record type">
    <button role="tab" aria-selected={kind==='person'} aria-controls="mdt-lookup-panel" id="mdt-person-tab" disabled={busy} onClick={()=>changeKind('person')} className={cx('btn min-h-11 flex-1 justify-center',kind==='person'&&'border-amber-500/60 bg-amber-500/15 text-amber-200')}><UserRound size={17} aria-hidden /> Persons</button>
    <button role="tab" aria-selected={kind==='vehicle'} aria-controls="mdt-lookup-panel" id="mdt-vehicle-tab" disabled={busy} onClick={()=>changeKind('vehicle')} className={cx('btn min-h-11 flex-1 justify-center',kind==='vehicle'&&'border-amber-500/60 bg-amber-500/15 text-amber-200')}><Car size={17} aria-hidden /> Vehicles</button>
   </div>
   <p className="text-xs leading-relaxed text-slate-400">LAS records are searched first, then EnformionGO on a person-search miss. New provider matches are saved locally. Searches are logged. These are possible record matches, not identity verification, warrant checks, or DMV/NCIC returns.</p>
   <div id="mdt-lookup-panel" role="tabpanel" aria-labelledby={kind==='person'?'mdt-person-tab':'mdt-vehicle-tab'}>
    <form onSubmit={search} className="space-y-3">
     <fieldset disabled={busy} className="space-y-3">
      {kind==='person'?<div className="grid gap-3 sm:grid-cols-[2fr_1fr]">
       <label className="label">Name or alias<input className="input h-11" autoFocus autoComplete="off" value={name} maxLength={100} onChange={e=>{setName(e.target.value);setResult(null)}} placeholder="Last, first or partial name" required minLength={2} /></label>
       <label className="label">Date of birth (optional)<input className="input h-11" type="date" value={dob} onChange={e=>{setDob(e.target.value);setResult(null)}} /></label>
      </div>:<>
       <div className="grid grid-cols-[2fr_1fr] gap-3">
        <label className="label">Plate<input className="input h-11 font-mono uppercase" autoFocus autoComplete="off" value={plate} maxLength={20} onChange={e=>{setPlate(e.target.value.toUpperCase());setResult(null)}} placeholder="Full or partial plate" /></label>
        <label className="label">State (optional)<input className="input h-11 font-mono uppercase" autoComplete="off" value={state} maxLength={2} onChange={e=>{setState(e.target.value.toUpperCase());setResult(null)}} placeholder="ND" /></label>
       </div>
       <label className="label">VIN (optional)<input className="input h-11 font-mono uppercase" autoComplete="off" value={vin} maxLength={17} onChange={e=>{setVin(e.target.value.toUpperCase());setResult(null)}} placeholder="Full VIN or at least four characters" /></label>
       <p className="text-xs text-slate-400">Enter at least two plate characters or four VIN characters. Provided fields must all match.</p>
      </>}
      <div className="flex flex-wrap items-end gap-3">
       <label className="label min-w-48 flex-1">Operational purpose<select className="input h-11" value={purpose} onChange={e=>setPurpose(e.target.value)}><option value="assigned-call">Assigned call</option><option value="patrol">Patrol duties</option><option value="follow-up">Incident follow-up</option></select></label>
       <button type="submit" className="btn btn-go h-11 px-5"><Search size={17} aria-hidden /> Search LAS records</button>
      </div>
     </fieldset>
    </form>
   </div>
   {busy&&<p role="status" className="text-sm text-slate-300">Searching LAS records, then EnformionGO if needed…</p>}
   {fbiQuery&&<FbiWanted query={fbiQuery} onCaution={text=>{if(request.current===fbiQuery.id)onFbiCaution?.(text)}}/>}
   {error&&<p role="alert" className="rounded-lg border border-rose-800 bg-rose-950/40 p-3 text-sm text-rose-200">{error}</p>}
   {result&&<section aria-label="Search results" className="space-y-3">
    {result.providerMessage&&<p role="status" className="rounded border border-amber-800/50 bg-amber-950/10 p-3 text-sm text-amber-200">{result.providerMessage}</p>}
    <p role="status" className="text-sm text-slate-300">{result.items.length===0?'No matching LAS records.':`${result.items.length}${result.more?'+':''} matching records`} <span className="text-xs text-slate-500">· {new Date(result.searchedAt).toLocaleTimeString([], {hour:'2-digit',minute:'2-digit'})}</span></p>
    {result.items.length===0&&<p className="text-xs text-slate-400">Try another spelling or refine the identifiers. No match does not establish that a person or vehicle has no history.</p>}
    {result.more&&<p className="text-xs text-amber-200">Showing the newest 50 matches. Add a birth date, plate state, or more characters to narrow the search.</p>}
    {result.items.map(hit=><article key={hit.incident+':'+hit.record} className="rounded-lg border border-slate-700 bg-slate-900 p-3">
     <div className="flex flex-wrap items-start gap-2"><h3 className="flex-1 text-base font-semibold text-slate-100">{result.kind==='person'?hit.name:[hit.plate||'No plate recorded',hit.state].filter(Boolean).join(' · ')}</h3><span className="chip border border-slate-600 text-xs text-slate-300">Local record · possible match</span></div>
     {result.kind==='person'?<><p className="mt-1 text-sm text-slate-300">DOB {hit.dob||'not recorded'}{hit.sex?` · Sex ${hit.sex}`:''}</p>{hit.description&&<p className="mt-1 text-sm text-slate-400">{hit.description}</p>}</>:<><p className="mt-1 text-sm text-slate-300">{[hit.color,hit.year,hit.make,hit.model,hit.body].filter(Boolean).join(' · ')||'Description not recorded'}</p><p className="mt-1 font-mono text-sm text-slate-400">VIN {hit.vin||'not recorded'}</p></>}
     {result.kind==='vehicle'&&hit.description&&<p className="mt-1 text-sm text-slate-400">{hit.description}</p>}
     <RecordDetailsDisplay person={hit.person} cautions={hit.cautions} ownership={hit.ownership} ownerName={hit.ownerName}/>{hit.person&&<PersonDetails person={hit.person}/>}
     {hit.registeredOwner&&<details className="mt-2 rounded border border-slate-600 p-2 text-sm"><summary className="cursor-pointer font-semibold text-amber-200">Registered owner · {hit.registeredOwner.name}</summary><p className="mt-2 text-xs text-slate-400">Linked local person record {hit.registeredOwner.record} · DOB {hit.registeredOwner.dob||'not recorded'}. Ownership is recorded information, not confirmation of the current driver.</p><RecordDetailsDisplay person={hit.registeredOwner.person}/>{hit.registeredOwner.person&&<PersonDetails person={hit.registeredOwner.person}/>} {hit.registeredOwner.flags?.map(f=><div key={f.id} className="mt-2 rounded border border-rose-700 p-2 text-rose-200"><b>Registered owner safety flag · {f.title}</b><p>{f.warning}</p><p>{f.instruction}</p></div>)}</details>}
     {hit.person?.agencyEmployee&&<p className="mt-2 rounded border border-teal-600 bg-teal-950/40 p-2 text-sm font-semibold text-teal-200">LAS employee · agency affiliation (informational)</p>}
     {hit.flags?.map(f=><div key={f.id} role="alert" className="mt-2 rounded border border-rose-700 bg-rose-950/40 p-2 text-sm text-rose-100"><b>Officer safety · {f.title}</b><p>{f.warning}</p><p>{f.instruction}</p><span className="font-mono text-xs">{f.id}</span></div>)}
     {!!hit.incidents?.length&&<p className="mt-2 text-xs text-amber-200">Connected incidents: {hit.incidents.map(r=>`${r.incident} (${r.role})`).join(' · ')}</p>}
     {hit.safetyConcern&&<p className="mt-2 flex items-center gap-2 rounded border border-amber-700 bg-amber-950/40 p-2 text-sm text-amber-200"><ShieldAlert size={16} aria-hidden />Recorded safety concern. Confirm current details with dispatch.</p>}
     <p className="mt-2 text-xs text-slate-500">{hit.source || 'LAS incident'} <span className="font-mono text-slate-300">{hit.incident || hit.record}</span> · recorded {new Date(hit.seen).toLocaleDateString()}</p>
    </article>)}
   </section>}
  </div>
 </Modal>
}
function PersonDetails({person:p}:{person:Person}){
 const fields:[string,string|undefined][]=[['First name',p.first],['Middle name',p.middle],['Last name',p.last],['Suffix',p.suffix],['Roles',p.roles?.join(', ')||p.role],['Location status',p.locKind],['SSN',p.ssn],['Licence',p.dl],['Licence state',p.dlState],['Other ID type',p.idType],['Other ID number',p.idNumber],['Other ID state',p.idState],['Address',p.address],['Building',p.building],['Floor',p.floor],['Unit',p.unit],['Phone',p.phone],['Approximate age',p.approxAge],['Race',p.race],['Ethnicity',p.ethnicity],['Height',p.height],['Weight',p.weight],['Build',p.build],['Hair',p.hair],['Eyes',p.eyes],['Clothing',p.clothing],['Scars, marks and tattoos',p.smt],['Aliases',p.aliases?.join(', ')],['Notes',p.notes]]
 return <details className="mt-2 rounded border border-slate-700 p-2 text-sm"><summary className="cursor-pointer font-semibold text-amber-200">Full person information</summary><dl className="mt-2 grid grid-cols-1 gap-2 sm:grid-cols-2">{fields.filter(([,v])=>v).map(([k,v])=><div key={k}><dt className="text-xs text-slate-500">{k}</dt><dd className="whitespace-pre-wrap text-slate-100">{v}</dd></div>)}</dl>{p.phones?.map((ph,i)=><p key={i}>{ph.type||'Phone'}: {ph.number}{ph.ext?` ext ${ph.ext}`:''} · {ph.source}{ph.verified?' · verified':''}</p>)}{p.safetyNote&&<section className="mt-2 rounded border border-rose-700 bg-rose-950/30 p-2"><b>{p.safetyNote.title}</b><p>{p.safetyNote.reason}</p><p>{p.safetyNote.weapons}</p><p>{p.safetyNote.instruction}</p><p className="text-xs text-slate-400">{p.safetyNote.source} · {p.safetyNote.reliability} · {p.safetyNote.scope}{p.safetyNote.flag?` · Flag ${p.safetyNote.flag}`:''}</p></section>}</details>
}
