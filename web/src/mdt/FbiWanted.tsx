import {useEffect,useRef,useState} from 'react'
import {ExternalLink,ShieldAlert} from 'lucide-react'
import {api,errText} from '../api'

export type FbiQuery={id:number;name:string;dob:string;purpose:string}
export type FbiNotice={uid:string;title:string;url:string;aliases?:string[];dates_of_birth_used?:string[];subjects?:string[];person_classification?:string;sex?:string;race?:string;status?:string;caution?:string;warning_message?:string;description?:string}
type FbiResult={items:FbiNotice[];checked:string;more:boolean;message?:string}
export function fbiAnnouncement(items:FbiNotice[]){return items.length?`Possible FBI Wanted notice match. Identity and current status are not confirmed. Verify with dispatch and the official FBI notice before acting. ${items.slice(0,3).map(n=>`${n.title}. Category ${(n.subjects||[]).join(', ')}. ${n.warning_message||''} ${n.caution||''}`).join('. ').slice(0,1200)}${items.length>3?' Additional possible matches appear on screen.':''}`:''}

export default function FbiWanted({query,onCaution}:{query:FbiQuery;onCaution?:(text:string)=>void}){
 const [result,setResult]=useState<FbiResult|null>(null),[error,setError]=useState('')
 const callback=useRef(onCaution);callback.current=onCaution
 useEffect(()=>{let active=true;setResult(null);setError('');api.post<FbiResult>('/mdt/records/fbi',{name:query.name,dob:query.dob,purpose:query.purpose}).then(r=>{if(!active)return;const items=(r.items||[]).filter(n=>typeof n.uid==='string'&&typeof n.title==='string');setResult({...r,items});if(items.length)callback.current?.(fbiAnnouncement(items))}).catch(e=>{if(active)setError(errText(e))});return()=>{active=false}},[query])
 return <section aria-label="FBI Wanted public notices" className="mt-4 space-y-2 rounded-lg border border-amber-700/60 bg-amber-950/10 p-3">
  <h3 className="flex items-center gap-2 text-sm font-bold text-amber-200"><ShieldAlert size={16}/> FBI Wanted public notices</h3>
  <p className="text-xs text-slate-400">Separate from LAS records and Enformion. Possible name matches, not confirmed identities or NCIC/warrant returns. Verify current status with dispatch and the official notice.</p>
  {error?<p role="status" className="text-sm text-amber-200">FBI lookup unavailable: {error} Local-record results remain available.</p>:!result?<p role="status" className="text-sm text-slate-400">Checking FBI public notices…</p>:<>
   <p className="text-xs text-slate-400">Checked {new Date(result.checked).toLocaleString()}{result.items.length===0?' · No candidate matches in the returned FBI notices.':''}</p>
   {(result.more||result.message)&&<p className="text-xs text-amber-200">{result.message||'Results are limited. Refine the name or check the official FBI site.'}</p>}
   {result.items.map(n=><article key={n.uid} role="alert" className="rounded border border-amber-600 bg-amber-950/30 p-3 text-sm">
    <h4 className="font-bold text-amber-100">Possible FBI Wanted notice match · {n.title}</h4>
    <p className="text-xs text-slate-300">{n.subjects?.join(' · ')}{n.person_classification?` · ${n.person_classification}`:''}</p>
    {!!n.aliases?.length&&<p>Aliases: {n.aliases.join(' · ')}</p>}
    <p>DOB: {n.dates_of_birth_used?.join(' · ')||'Not published'}{n.sex?` · ${n.sex}`:''}{n.race?` · ${n.race}`:''}</p>
    {n.description&&<p className="mt-1">{n.description}</p>}{n.warning_message&&<p className="mt-2 font-bold text-amber-100">{n.warning_message}</p>}{n.caution&&<p className="mt-2 text-amber-100">FBI caution: {n.caution}</p>}
    <a href={n.url} target="_blank" rel="noopener noreferrer" className="mt-2 inline-flex items-center gap-1 text-sky-300 underline">Official FBI notice <ExternalLink size={13}/></a>
   </article>)}
  </>}
 </section>
}
