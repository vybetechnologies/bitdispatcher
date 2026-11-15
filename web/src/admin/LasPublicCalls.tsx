import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { api, errText } from '../api'
import { Page, Table } from './Admin'
type Item = { incident:string; type:string; status:string; eligible:boolean; withheld:boolean }
type Preview = { seq:number; items:Item[] }
export default function LasPublicCalls() {
 const [v,setV]=useState<Preview|null>(null)
 const load=()=>api.get<Preview>('/admin/las-public-calls').then(setV).catch(e=>toast.error(errText(e)))
 useEffect(()=>{load()},[])
 const setWithheld=async(item:Item)=>{if(!v)return;try{await api.put('/admin/las-public-calls/'+encodeURIComponent(item.incident),{seq:v.seq,withheld:!item.withheld});await load();toast.success(item.withheld?'Automatic public visibility restored':'Call withheld from the public feed')}catch(e){toast.error(errText(e));await load()}}
 return <Page title="Public calls feed" intro="Eligible calls publish automatically. Administrators can withhold an individual call when confidentiality requires it.">
 <section className="panel mb-4 space-y-3 rounded-lg p-4 text-sm">
 <p>The public sees an opaque reference, broad category, approximate time, Active/Closed status, broad area and a hundred-block street when safely recognizable. Names, narratives, clients, units, coordinates, exact addresses and attachments are excluded.</p>
 <p>Publication starts after five minutes. Received times are rounded to 15 minutes; history covers up to 90 days and the newest 1,000 entries. Welfare, emergencies, follow-ups, other/unknown and unrecognized call types are excluded by default. Ambiguous locations are withheld.</p>
 <a className="btn inline-flex" href="https://lakesareasecurity.com/calls" target="_blank" rel="noopener noreferrer">Open public calls page</a>
 </section>
 {!v?<p>Loading…</p>:v.items.length===0?<p>No incidents yet. The automatic feed is connected; no sample calls are published.</p>:<Table head={['Incident','Type','Status','Public visibility','Action']}>{v.items.map(item=><tr key={item.incident}><td>{item.incident}</td><td>{item.type}</td><td>{item.status}</td><td>{!item.eligible?'Excluded type':item.withheld?'Withheld':'Automatic (after delay)'}</td><td>{item.eligible&&<button className="btn" onClick={()=>setWithheld(item)}>{item.withheld?'Restore automatic visibility':'Withhold for confidentiality'}</button>}</td></tr>)}</Table>}
 </Page>
}
