import {useEffect,useRef,useState} from 'react'
import {api,errText} from '../api'

type Choice={id:string;name:string;dob?:string;archived?:boolean}
/** Local administrative lookup: does not spend an external-provider search. */
export default function RegisteredOwnerPicker({value,name,onChange,title='Registered owner (optional person record)',selectLabel='Link as registered owner',searchLabel='Find registered owner',helpText='Choose an existing local person, or leave blank. Vehicle affiliation and safety flags are managed separately.'}:{value?:string;name?:string;onChange:(id:string,name:string)=>void;title?:string;selectLabel?:string;searchLabel?:string;helpText?:string}){
 const [query,setQuery]=useState(''),[items,setItems]=useState<Choice[]>([]),[busy,setBusy]=useState(false),[error,setError]=useState('')
 const request=useRef(0)
 useEffect(()=>()=>{request.current++},[])
 const search=async()=>{
  if(query.trim().length<2||busy)return
  const id=++request.current;setBusy(true);setError('')
  try{const result=await api.get<{items:Choice[]}>(`/admin/las-records?kind=person&q=${encodeURIComponent(query.trim())}`);if(id===request.current)setItems(result.items.filter(p=>!p.archived))}
  catch(e){if(id===request.current)setError(errText(e))}
  finally{if(id===request.current)setBusy(false)}
 }
 return <fieldset className="space-y-2 rounded border border-slate-700 p-3"><legend className="label">{title}</legend>
  <p className="text-sm">{value?`Linked person: ${name||value}${name?` · ${value}`:''}`:'No person record linked.'}</p>
  {value&&<button type="button" className="btn" onClick={()=>onChange('','')}>Remove registered-owner link</button>}
  <div className="flex flex-wrap gap-2"><input className="input min-w-0 flex-1" aria-label={searchLabel} placeholder="Person name or local record ID" maxLength={100} value={query} onChange={e=>{request.current++;setQuery(e.target.value);setItems([]);setBusy(false)}}/><button type="button" className="btn" disabled={busy||query.trim().length<2} onClick={()=>void search()}>{busy?'Searching…':'Find local person'}</button></div>
  {error&&<p role="alert" className="text-sm text-rose-300">{error}</p>}
  {items.map(p=><div key={p.id} className="flex flex-wrap items-center gap-2 rounded bg-slate-950/60 p-2 text-sm"><span className="flex-1">{p.name} · {p.dob||'DOB not recorded'} · {p.id}</span><button type="button" className="btn" onClick={()=>{onChange(p.id,p.name);setItems([]);setQuery('')}}>{selectLabel}</button></div>)}
  <p className="text-xs text-slate-400">{helpText}</p>
 </fieldset>
}
