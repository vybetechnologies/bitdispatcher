import {useState} from 'react'
import {Plus,CheckCircle2} from 'lucide-react'
import {toast} from 'sonner'
import {api,errText} from '../api'
import {Modal} from '../ui'
import type {MDTView} from './Mdt'

export default function IncidentActions({m,onSaved}:{m:MDTView;onSaved:()=>void}){
 const [mode,setMode]=useState<'create'|'close'|null>(null),[busy,setBusy]=useState(false),[error,setError]=useState('')
 const [type,setType]=useState(''),[priority,setPriority]=useState(''),[address,setAddress]=useState(''),[narrative,setNarrative]=useState(''),[gps,setGps]=useState(false),[disposition,setDisposition]=useState('')
 const close=()=>{if(!busy){setMode(null);setError('')}}
 const submit=async(e:React.FormEvent)=>{e.preventDefault();if(busy)return;setBusy(true);setError('');try{
  const result=mode==='create'?await api.post<{queued?:boolean;number?:string}>('/mdt/incidents',{type,priority:priority?Number(priority):0,narrative,location:{address,...(gps&&m.unit.position?{lat:m.unit.position.lat,lon:m.unit.position.lon}:{})}}):await api.post<{queued?:boolean}>('/mdt/close',{incident:m.incident?.number,disposition,text:narrative})
  toast.success(result?.queued?'Saved on this device · pending dispatch sync':mode==='create'?'Incident created and assigned to your unit':'Incident closed; your unit is available')
  setMode(null);setNarrative('');onSaved()
 }catch(e){setError(errText(e))}finally{setBusy(false)}}
 return <>
  <button className="btn" disabled={!!m.unit.incident||m.unit.status!=='available'} title="Create and assign an officer-initiated incident to your unit" onClick={()=>{setMode('create');setNarrative('');setError('')}}><Plus size={15}/> New incident</button>
  {m.incident&&<button className="btn" onClick={()=>{setMode('close');setNarrative('');setError('')}}><CheckCircle2 size={15}/> Close incident</button>}
  <Modal open={mode!==null} onClose={close} title={mode==='create'?'Create officer-initiated incident':'Close your incident'}>
   <form onSubmit={submit} className="space-y-3"><fieldset disabled={busy} className="space-y-3">
    {mode==='create'?<>
     <p className="text-sm text-slate-400">This call will be assigned to {m.unit.id} and appear in CAD.</p>
     <label className="label">Call type<select required className="input" value={type} onChange={e=>setType(e.target.value)}><option value="">Choose call type</option>{[...(m.callTypes||[])].sort((a,b)=>a.name.localeCompare(b.name)).map(t=><option key={t.code} value={t.code}>{t.name} ({t.code})</option>)}</select></label>
     <label className="label">Priority<select className="input" value={priority} onChange={e=>setPriority(e.target.value)}><option value="">Call type default</option>{[1,2,3,4,5].map(p=><option key={p}>{p}</option>)}</select></label>
     <label className="label">Location<input required={!gps} className="input" value={address} maxLength={500} onChange={e=>setAddress(e.target.value)} placeholder="Address, business or location description"/></label>
     <label className="flex items-center gap-2 text-sm"><input type="checkbox" disabled={!m.unit.position} checked={gps} onChange={e=>setGps(e.target.checked)}/> Include current unit GPS position{m.unit.position?.at&&` · ${new Date(m.unit.position.at).toLocaleTimeString()}`}</label>
    </>:<><p className="text-sm text-slate-300">Close {m.incident?.number}. Your unit will become available. Incidents with other active units must be closed by dispatch.</p><label className="label">Disposition<select required className="input" value={disposition} onChange={e=>setDisposition(e.target.value)}><option value="">Choose disposition</option>{(m.dispositions||[]).map(d=><option key={d}>{d}</option>)}</select></label></>}
    <label className="label">{mode==='create'?'Initial narrative':'Closing note'}<textarea className="input min-h-24" value={narrative} maxLength={10000} onChange={e=>setNarrative(e.target.value)}/></label>
    {error&&<p role="alert" className="text-sm text-rose-200">{error}</p>}
    <div className="flex justify-end gap-2"><button type="button" className="btn" onClick={close}>Cancel</button><button className="btn btn-go" disabled={busy}>{busy?'Saving…':mode==='create'?'Create and assign to me':'Close incident'}</button></div>
   </fieldset></form>
  </Modal>
 </>
}
