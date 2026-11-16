import { useEffect, useState } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import { FilePlus2, X } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { api, errText } from '../api'
import { isWindowsMdt } from '../desktop'
import type { Report } from '../rms/Rms'

/** Non-modal so the unit controls and emergency dock remain available. */
export default function MdtReports({onClose}:{onClose:()=>void}) {
 const [reports,setReports]=useState<Report[]>([])
 const [filter,setFilter]=useState('')
 const [error,setError]=useState('')
 const nav=useNavigate()
 useEffect(()=>{let active=true;api.get<Report[]>('/rms/reports').then(r=>{if(active)setReports(r)}).catch(e=>{if(active)setError(errText(e))});return()=>{active=false}},[])
 const open=(number:string)=>nav(`${isWindowsMdt()?'/mdt/reports':'/rms'}/${number}`)
 return <Dialog.Root open modal={false} onOpenChange={o=>{if(!o)onClose()}}>
  <Dialog.Content className="mdt-reports-drawer" aria-describedby={undefined}>
   <header><Dialog.Title>My reports</Dialog.Title><button className="btn" onClick={async()=>{try{const r=await api.post<{report:Report}>('/rms/reports',{offenses:[],people:[],property:[],vehicles:[]});open(r.report.number)}catch(e){toast.error(errText(e))}}}><FilePlus2 size={14}/> New</button><Dialog.Close className="btn" aria-label="Close reports"><X size={16}/></Dialog.Close></header>
   <div className="mdt-report-filters" aria-label="Filter reports">{['','draft','submitted','returned','approved'].map(f=><button key={f} className="btn" aria-pressed={filter===f} onClick={()=>setFilter(f)}>{f||'all'}</button>)}</div>
   <div className="mdt-report-list">{error&&<p role="alert">{error}</p>}{reports.filter(r=>!filter||r.status===filter).map(r=><button key={r.number} className="mdt-report-item" onClick={()=>open(r.number)}><b>{r.number}</b><span>{r.status}</span><p>{r.location||'No location'} · {r.incident||'No linked call'}</p>{!!r.problems&&<small>{r.problems} to fix</small>}<time>{new Date(r.updated).toLocaleString()}</time></button>)}{!error&&!reports.filter(r=>!filter||r.status===filter).length&&<p>No reports in this view.</p>}</div>
   <footer><p>Drafts save as you type. Open a report to view or edit all its fields.</p></footer>
  </Dialog.Content>
 </Dialog.Root>
}
