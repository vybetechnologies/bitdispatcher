import type { Session } from './types'
import type { MDTView } from './mdt/Mdt'

declare global {
 interface Window { lasDesktop?:{version:string;call:(op:string,key?:string,value?:unknown)=>Promise<unknown>} }
}
export const isWindowsMdt=()=>!!window.lasDesktop
export type DesktopCommand={ID:string;Owner:string;Unit:string;Method:string;Path:string;Body:Record<string,unknown>;Incident:string;BaseStatusAt:string;At:string;error?:string;status?:number;localReport?:string;Binary?:string;FileName?:string;Caption?:string}
export type DesktopState={connected:boolean;pending:number;lastSync:number;error:string;preparing:boolean}
let state:DesktopState={connected:true,pending:0,lastSync:0,error:'',preparing:false}
let identity:{session:Session}|null=null
let initialized:Promise<void>|null=null
let writeChain=Promise.resolve()
let flushing:Promise<void>|null=null
let preparation=0
const results=new Map<string,Response>()
const awaitingResult=new Set<string>()
const reply=(data:unknown,status=200)=>new Response(status===204?null:JSON.stringify(data),{status,headers:{'Content-Type':'application/json'}})
export const desktopState=()=>state
function announce(update:Partial<DesktopState>){
 if(Object.entries(update).every(([key,value])=>state[key as keyof DesktopState]===value))return
 if(update.connected===true&&!state.connected)preparation=0
 state={...state,...update};window.dispatchEvent(new Event('las:desktop'))
 void window.lasDesktop?.call('status','',`LAS MDT · ${state.connected?'Connected':'OFFLINE — radio/phone for urgent help'} · ${state.pending} pending${state.error?' · review needed':''}`).catch(()=>{})
}
async function read<T>(key:string):Promise<T|null>{const value=await window.lasDesktop!.call('read',key);return typeof value==='string'?JSON.parse(value) as T:null}
async function save(key:string,value:unknown){await window.lasDesktop!.call('write',key,JSON.stringify(value))}
const scope=()=>identity?.session.username+'|'+identity?.session.mdtUnit
const queueKey=()=> 'queue:'+scope()
const commandKey=(id:string)=>'command:'+scope()+':'+id
const cacheKey=(path:string)=>'cache:'+scope()+':'+path
async function init(){if(!initialized)initialized=(async()=>{identity=await read<{session:Session}>('identity');await refreshCount()})();await initialized}
async function ids(){return await read<string[]>(queueKey())||[]}
export async function desktopOutbox(){if(!isWindowsMdt())return [];await init();const list:DesktopCommand[]=[];for(const id of await ids()){const item=await read<DesktopCommand>(commandKey(id));if(item)list.push(item)}return list}
async function refreshCount(){if(identity)announce({pending:(await ids()).length});else announce({pending:0})}
async function remove(id:string){await save(queueKey(),(await ids()).filter(x=>x!==id));await window.lasDesktop!.call('delete',commandKey(id));await refreshCount()}
export async function discardDesktopCommand(id:string){await init();await serialize(async()=>remove(id));announce({error:''})}
export async function retryDesktopSync(){announce({error:''});await syncDesktop()}
function serialize<T>(f:()=>Promise<T>):Promise<T>{const result=writeChain.then(f);writeChain=result.then(()=>{},()=>{});return result}
async function network(method:string,path:string,body?:unknown){
 const headers:Record<string,string>={'X-BitDispatcher':'1'}
 if(body!==undefined)headers['Content-Type']='application/json'
 return fetch('./api'+path,{method,headers,body:body===undefined?undefined:JSON.stringify(body),credentials:'same-origin',signal:AbortSignal.timeout(12000)})
}
async function cached<T>(path:string){return read<{at:number;data:T}>(cacheKey(path))}
async function putCache(path:string,data:unknown){await save(cacheKey(path),{at:Date.now(),data})}
async function project(data:MDTView){
 const m=structuredClone(data) as MDTView&{offlineStatusAt?:string}
 for(const cmd of await desktopOutbox()){
  if(cmd.error)continue
  const b=cmd.Body
  if(cmd.Path==='/mdt/status'){
   m.unit.status=String(b.status);m.unit.statusAt=cmd.At;m.offlineStatusAt='local:'+cmd.ID
   if(['available','oos'].includes(m.unit.status)){m.unit.incident=undefined;m.incident=undefined}
  }else if(cmd.Path==='/mdt/markout'){m.unit.status='markout';m.unit.statusAt=cmd.At;m.offlineStatusAt='local:'+cmd.ID;m.unit.incident=undefined;m.incident=undefined}
  else if(cmd.Path==='/mdt/attach'){
   const call=m.pending.find(x=>x.number===b.incident);if(call){m.incident=call;m.unit.incident=call.number;m.unit.status='dispatched';m.unit.statusAt=cmd.At;m.offlineStatusAt='local:'+cmd.ID;m.pending=m.pending.filter(x=>x.number!==call.number)}
  }else if(cmd.Path==='/mdt/note'&&m.incident?.number===cmd.Incident){
   m.incident.timeline=[...(m.incident.timeline||[]),{kind:'note',text:String(b.text)+' (saved offline — pending sync)',at:cmd.At,actor:cmd.Owner,seq:0}]
  }else if(cmd.Path==='/mdt/message'){
   m.messages=[...(m.messages||[]),{id:'OFFLINE-'+cmd.ID,at:cmd.At,from:cmd.Unit,fromUnit:true,to:b.to as string[]||['DISPATCH'],text:String(b.text)+' (pending sync)',incident:cmd.Incident}]
  }else if(cmd.Path==='/mdt/emergency'){m.emergency={key:'OFFLINE-'+cmd.ID,raised:cmd.At}}
  else if(cmd.Path==='/mdt/position'){m.unit.position={...b,at:cmd.At,source:'mdt-offline'} as MDTView['unit']['position']}
 }
 return m
}
async function prepare(){
 if(!identity?.session.mdtUnit||Date.now()-preparation<300000)return
 preparation=Date.now();announce({preparing:true})
 try{
  for(const path of ['/mdt/offline/records','/rms/meta','/rms/reports','/comms?folder=inbox','/roads']){
   const response=await network('GET',path);if(response.ok){const data:unknown=await response.json();await putCache(path,data)
    if(path==='/rms/reports'&&Array.isArray(data)){for(const report of data.slice(0,100)){const r=report as {number:string};const detail=await network('GET','/rms/reports/'+encodeURIComponent(r.number));if(detail.ok)await putCache('/rms/reports/'+r.number,await detail.json())}}
   }
  }
  const inbox=await cached<{items:{id:string}[]}>('/comms?folder=inbox')
  for(const item of inbox?.data.items.slice(0,100)||[]){
   const detail=await network('GET','/comms/'+encodeURIComponent(item.id))
   if(detail.ok){const data=await detail.json() as {files?:{id:string}[]};await putCache('/comms/'+item.id,data);for(const file of data.files||[])void window.lasDesktop!.call('cacheFile','/api/comms/files/'+file.id).catch(()=>{})}
  }
  if(!await read('map.ready'))await window.lasDesktop!.call('downloadMap')
 }catch(e){announce({error:'Offline preparation is incomplete: '+(e instanceof Error?e.message:String(e))})}
 finally{announce({preparing:false})}
}
function allowed(method:string,path:string){
 return method==='PUT'&&path==='/mdt/crew'||method==='POST'&&(['/mdt/status','/mdt/markout','/mdt/note','/mdt/position','/mdt/emergency','/mdt/message','/mdt/attach','/rms/reports'].includes(path)||/^\/mdt\/message\/[A-Za-z0-9-]+\/read$/.test(path)||/^\/comms\/[A-Za-z0-9_-]+\/(open|ack)$/.test(path)||/^\/incidents\/[A-Za-z0-9_-]+\/files$/.test(path))
}
async function queueWrite(method:string,path:string,body:Record<string,unknown>,extra:Partial<DesktopCommand>={}){
 if(!identity?.session.mdtUnit)throw new Error('Sign in online with your assigned MDT unit before working offline.')
 const cachedMdt=await cached<MDTView>('/mdt');if(!cachedMdt)throw new Error('Open the MDT online once before recording offline actions.')
 const m=await project(cachedMdt.data);const id=crypto.randomUUID()
 const cmd:DesktopCommand={ID:id,Owner:identity.session.username,Unit:identity.session.mdtUnit,Method:method,Path:path,Body:structuredClone(body),Incident:m.unit.incident||'',BaseStatusAt:m.offlineStatusAt||m.unit.statusAt,At:new Date().toISOString(),...extra}
 if(path==='/rms/reports'&&!body.number){cmd.localReport='OFFLINE-'+id}
 await save(commandKey(id),cmd);await save(queueKey(),[...await ids(),id]);await refreshCount()
 if(path==='/rms/reports'){
  const number=String(body.number||cmd.localReport);const report={...body,number,status:body.status||'draft',author:identity.session.username,updated:body.updated||cmd.At,created:body.created||cmd.At,offenses:body.offenses||[],people:body.people||[],property:body.property||[],vehicles:body.vehicles||[]}
  await putCache('/rms/reports/'+number,{report,checks:[],agency:[]})
  const list=await cached<Record<string,unknown>[]>('/rms/reports');await putCache('/rms/reports',[...(list?.data||[]).filter(x=>x.number!==number),report])
 }
 return cmd
}
async function queuedResponse(cmd:DesktopCommand){
 if(cmd.Path==='/rms/reports'){return reply((await cached('/rms/reports/'+String(cmd.Body.number||cmd.localReport)))?.data)}
 return reply({queued:true,id:'OFFLINE-'+cmd.ID,key:'OFFLINE-'+cmd.ID})
}
export async function syncDesktop(){
 if(!isWindowsMdt())return
 await init();if(flushing)return flushing
 flushing=(async()=>{
  if(!identity?.session.mdtUnit)return
  try{
   const meResponse=await network('GET','/auth/me');if(!meResponse.ok)throw new Error('Sign in again to sync saved work.')
   const me=await meResponse.json() as {signedIn:boolean;session?:Session}
   if(!me.signedIn||me.session?.username!==identity.session.username||me.session.mdtUnit!==identity.session.mdtUnit){announce({error:'Saved work is bound to its original account and unit. Sign in with that account to sync.'});return}
   announce({connected:true})
   for(;;){
    const list=await desktopOutbox();const cmd=list.find(x=>x.Path==='/mdt/emergency')||list[0];if(!cmd)break
    const mappings=await read<Record<string,{number:string;updated:string}>>('reports:'+scope())||{}
    const payload=structuredClone(cmd);delete payload.error;delete payload.status
    if(cmd.Path==='/rms/reports'){
     const alias=String(cmd.Body.number||'');if(mappings[alias]){payload.Body.number=mappings[alias].number;payload.Body.updated=mappings[alias].updated}
    }
    const response=await network('POST','/mdt/offline/sync',payload)
    if(!response.ok){
     const error=await response.clone().json().catch(()=>({})) as {message?:string}
     cmd.error=error.message||'Sync refused this action';cmd.status=response.status;await save(commandKey(cmd.ID),cmd);if(awaitingResult.has(cmd.ID))results.set(cmd.ID,response);announce({error:cmd.error});break
    }
    if(awaitingResult.has(cmd.ID))results.set(cmd.ID,response.clone())
    if(cmd.Path==='/rms/reports'){
     const result=await response.clone().json() as {report?:{number:string;updated:string};appliedUpdated?:string}
     if(result.report){const value={number:result.report.number,updated:result.appliedUpdated||result.report.updated};mappings[String(cmd.Body.number||cmd.localReport)]=value;mappings[value.number]=value;await save('reports:'+scope(),mappings);await putCache('/rms/reports/'+value.number,result)}
    }
    await remove(cmd.ID)
    if(['/mdt/status','/mdt/markout','/mdt/attach','/mdt/crew'].includes(cmd.Path)){try{const fresh=await network('GET','/mdt');if(fresh.ok)await putCache('/mdt',await fresh.json())}catch{/* The command receipt remains valid when the follow-up view cannot load. */}}
    announce({lastSync:Date.now(),error:''});window.dispatchEvent(new Event('las:synced'))
   }
  }catch(e){announce({connected:false,error:e instanceof Error?e.message:'Connection unavailable'})}
 })().finally(()=>{flushing=null})
 return flushing
}
async function offlineLookup(body:Record<string,unknown>){
 const snapshot=await cached<{records:Record<string,unknown>[];at:string}>('/mdt/offline/records')
 if(body.kind==='person'&&String(body.name||'').trim().length<2)throw new Error('Enter at least two name characters.')
 if(body.kind==='vehicle'&&String(body.plate||'').replace(/[^a-z0-9]/gi,'').length<2&&String(body.vin||'').length<4)throw new Error('Enter two plate characters or four VIN characters.')
 if(!snapshot)throw new Error('Local records have not been downloaded to this device. Reconnect to prepare them.')
 const normal=(v:unknown)=>String(v||'').toUpperCase().replace(/[^A-Z0-9]/g,'')
 const words=String(body.name||'').toUpperCase().split(/[^A-Z0-9]+/).filter(Boolean)
 const items=snapshot.data.records.filter(r=>{
  if(body.kind==='person'){const p=r.person as {aliases?:string[]}|undefined;return !!p&&words.every(word=>[r.name,...p.aliases||[]].join(' ').toUpperCase().includes(word))&&(!body.dob||r.dob===body.dob)}
  return !r.person&&(!body.plate||normal(r.plate).includes(normal(body.plate)))&&(!body.vin||normal(r.vin).includes(normal(body.vin)))&&(!body.state||normal(r.state)===normal(body.state))
 })
 return reply({kind:body.kind,items:items.slice(0,50),more:items.length>50,source:'Downloaded LAS records',searchedAt:new Date().toISOString(),providerMessage:'Offline local records as of '+snapshot.data.at+'. Enformion searches require a connection.'})
}
export async function desktopRequest(method:string,path:string,body?:unknown):Promise<Response>{
 if(!isWindowsMdt())return network(method,path,body)
 await init()
 if(path==='/auth/logout'){
  try{await network(method,path,body)}catch{/* Local sign-out also clears the device's browser credentials. */}
  identity=null;await window.lasDesktop!.call('delete','identity');await window.lasDesktop!.call('clearCookies');announce({pending:0});return reply(undefined,204)
 }
 if(allowed(method,path))return serialize(async()=>{
  const cmd=await queueWrite(method,path,(body||{}) as Record<string,unknown>)
  if(state.connected){awaitingResult.add(cmd.ID);try{await syncDesktop()}finally{awaitingResult.delete(cmd.ID)}}
  const result=results.get(cmd.ID);results.delete(cmd.ID);return result||queuedResponse(cmd)
 })
 if(method==='GET'&&path.startsWith('/rms/reports/OFFLINE-')){
  const number=decodeURIComponent(path.slice('/rms/reports/'.length));const mappings=await read<Record<string,{number:string}>>('reports:'+scope())||{}
  if(mappings[number])return desktopRequest('GET','/rms/reports/'+mappings[number].number)
  const draft=await cached(path);if(draft)return reply(draft.data)
 }
 try{
  const response=await network(method,path,body);announce({connected:true})
  if(path==='/auth/me'&&response.ok){
   const data=await response.clone().json() as {signedIn:boolean;session?:Session}
   if(data.signedIn&&data.session){identity={session:data.session};await save('identity',identity);await refreshCount()}else if(!data.signedIn){identity=null;await window.lasDesktop!.call('delete','identity')}
  }
  if(method==='GET'&&response.ok&&identity){
   const data:unknown=await response.clone().json().catch(()=>null)
   if(data!==null){await putCache(path,data);if(path==='/mdt'){void prepare();const m=data as MDTView;for(const file of m.incident?.files||[])if(!file.hidden)void window.lasDesktop!.call('cacheFile','/api/incidents/'+m.incident!.number+'/files/'+file.id).catch(()=>{});return reply(await project(m))}}
  }
  return response
 }catch(error){
  announce({connected:false})
  if(path==='/auth/me'&&identity)return reply({signedIn:true,session:identity.session})
  if(path==='/mdt/records/search')return offlineLookup((body||{}) as Record<string,unknown>)
  if(path==='/map/routes'){
   const map=await window.lasDesktop!.call('map','/api/map/routes',{...body as Record<string,unknown>,roads:(await cached('/roads'))?.data}) as {Status:number;Data:string}
   if(map.Status===200)return new Response(Uint8Array.from(atob(map.Data),c=>c.charCodeAt(0)),{headers:{'Content-Type':'application/json'}})
  }
  if(method==='GET'&&identity){const value=await cached(path);if(value)return reply(path==='/mdt'?await project(value.data as MDTView):path==='/state'?{...value.data as Record<string,unknown>,serverTime:new Date().toISOString()}:value.data)}
  if(path==='/auth/active')return reply(undefined,204)
  throw new Error('Offline: this information is not downloaded yet. Your saved work remains on this device.',{cause:error})
 }
}
export async function desktopFile(path:string,file:File):Promise<Response>{
 if(!isWindowsMdt())return fetch('./api'+path,{method:'POST',body:file,headers:{'X-BitDispatcher':'1'},credentials:'same-origin'})
 await init();if(file.size>25*1024*1024)throw new Error('Files must be 25 MB or smaller.')
 const url=new URL(path,location.origin);const bytes=new Uint8Array(await file.arrayBuffer());let raw='';for(let at=0;at<bytes.length;at+=8192)raw+=String.fromCharCode(...bytes.subarray(at,at+8192))
 return serialize(async()=>{const cmd=await queueWrite('POST',url.pathname,{}, {Binary:btoa(raw),FileName:file.name,Caption:url.searchParams.get('caption')||''});if(state.connected)await syncDesktop();const result=results.get(cmd.ID);results.delete(cmd.ID);return result||queuedResponse(cmd)})
}
export function startDesktopSync(){
 if(!isWindowsMdt())return ()=>{}
 const run=()=>void syncDesktop();const timer=window.setInterval(run,5000)
 window.addEventListener('online',run);window.addEventListener('las:sync',run)
 return()=>{clearInterval(timer);window.removeEventListener('online',run);window.removeEventListener('las:sync',run)}
}
