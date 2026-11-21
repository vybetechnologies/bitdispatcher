import { afterEach, describe, expect, it, vi } from 'vitest'

afterEach(()=>{delete window.lasDesktop;vi.unstubAllGlobals();vi.resetModules()})
function storageFixture(){
 const files=new Map<string,string>()
 files.set('identity',JSON.stringify({session:{username:'alex',name:'Alex',role:'field',duty:'field',mdtUnit:'U1'}}))
 files.set('cache:alex|U1:/mdt',JSON.stringify({at:Date.now(),data:{serverTime:new Date().toISOString(),unit:{id:'U1',status:'available',statusAt:'2026-11-15T12:00:00Z'},pending:[],partners:[],statuses:['available','oos'],settings:{},health:{},messages:[]}}))
 window.lasDesktop={version:'1.0.0',call:vi.fn(async(op,key,value)=>{
  if(op==='read')return files.get(key||'')||null
  if(op==='write'){files.set(key||'',String(value));return null}
  if(op==='delete'){files.delete(key||'');return null}
  return null
 })}
 return files
}
describe('Windows MDT durable offline work',()=>{
 it('maps an offline-created incident to its server number for notes and closure',async()=>{
  storageFixture();vi.stubGlobal('fetch',vi.fn().mockRejectedValue(new TypeError('offline')))
  const desktop=await import('./desktop')
  const created=await(await desktop.desktopRequest('POST','/mdt/incidents',{type:'PATROL',location:{address:'Synthetic location'},narrative:'Synthetic call'})).json()
  expect(created.number).toMatch(/^OFFLINE-/)
  expect((await(await desktop.desktopRequest('GET','/mdt')).json()).incident.number).toBe(created.number)
  await desktop.desktopRequest('POST','/mdt/note',{text:'Synthetic note'})
  await desktop.desktopRequest('POST','/mdt/close',{incident:created.number,disposition:'Resolved',text:'Synthetic close'})
  const sent:Record<string,unknown>[]=[]
  vi.stubGlobal('fetch',vi.fn(async(url:string,options?:RequestInit)=>{
   if(url.endsWith('/auth/me'))return new Response(JSON.stringify({signedIn:true,session:{username:'alex',mdtUnit:'U1'}}))
   if(url.endsWith('/offline/sync')){const payload=JSON.parse(String(options?.body));sent.push(payload);return payload.Path==='/mdt/incidents'?new Response(JSON.stringify({number:'PERMANENT-CALL'})):new Response(null,{status:204})}
   return new Response(JSON.stringify({unit:{id:'U1',status:'available',statusAt:'2026-11-17T12:00:00Z'},pending:[],messages:[],settings:{},health:{}}))
  }))
  await desktop.syncDesktop();expect(await desktop.desktopOutbox()).toHaveLength(0)
  expect(sent[1].Incident).toBe('PERMANENT-CALL');expect(sent[2].Incident).toBe('PERMANENT-CALL')
  expect((sent[2].Body as {incident:string}).incident).toBe('PERMANENT-CALL')
 })
 it('queues deletion of an offline report and resolves its assigned server number',async()=>{
  storageFixture();vi.stubGlobal('fetch',vi.fn().mockRejectedValue(new TypeError('offline')))
  const desktop=await import('./desktop')
  const draft=await(await desktop.desktopRequest('POST','/rms/reports',{narrative:'Synthetic draft'})).json()
  await desktop.desktopRequest('DELETE','/rms/reports/'+draft.report.number)
  const sent:{Path:string;Method:string}[]=[]
  vi.stubGlobal('fetch',vi.fn(async(url:string,options?:RequestInit)=>{
   if(url.endsWith('/auth/me'))return new Response(JSON.stringify({signedIn:true,session:{username:'alex',mdtUnit:'U1'}}))
   const command=JSON.parse(String(options?.body));sent.push(command)
   return command.Method==='DELETE'?new Response(null,{status:204}):new Response(JSON.stringify({report:{number:'R-PERM',updated:'2026-11-17T12:00:00Z'}}))
  }))
  await desktop.syncDesktop();expect(await desktop.desktopOutbox()).toHaveLength(0)
  expect(sent[1]).toMatchObject({Method:'DELETE',Path:'/rms/reports/R-PERM'})
 })
 it('retains a rejected legacy entry while syncing unrelated new work',async()=>{
  const files=storageFixture();vi.stubGlobal('fetch',vi.fn().mockRejectedValue(new TypeError('offline')))
  const desktop=await import('./desktop')
  await desktop.desktopRequest('POST','/mdt/position',{lat:46.87,lon:-96.79})
  await desktop.desktopRequest('POST','/mdt/position',{lat:46.88,lon:-96.79})
  const [legacy,newer]=await desktop.desktopOutbox()
  const key='command:alex|U1:'+legacy.ID
  const old=JSON.parse(files.get(key)!);delete old.Context;files.set(key,JSON.stringify(old))
  const sent:string[]=[]
  vi.stubGlobal('fetch',vi.fn(async(url:string,options?:RequestInit)=>{
   if(url.endsWith('/auth/me'))return new Response(JSON.stringify({signedIn:true,session:{username:'alex',mdtUnit:'U1'}}))
   const command=JSON.parse(String(options?.body));sent.push(command.ID)
   return command.ID===legacy.ID?new Response(JSON.stringify({message:'Older app entry requires review'}),{status:409}):new Response(null,{status:204})
  }))
  await desktop.syncDesktop()
  expect(sent).toEqual([legacy.ID,newer.ID]);expect(await desktop.desktopOutbox()).toHaveLength(1)
  expect((await desktop.desktopOutbox())[0].error).toContain('requires review')
  sent.length=0;await desktop.syncDesktop();expect(sent).toEqual([])
  await desktop.retryDesktopSync();expect(sent).toEqual([legacy.ID])
  vi.stubGlobal('fetch',vi.fn(async(url:string)=>url.endsWith('/auth/me')?new Response(JSON.stringify({signedIn:true,session:{username:'alex',mdtUnit:'U1'}})):new Response(null,{status:204})))
  await desktop.recoverLegacyLiveEntries()
  expect(await desktop.desktopOutbox()).toHaveLength(0)
  expect(JSON.parse(files.get('legacy-backup:alex|U1:'+legacy.ID)!).ID).toBe(legacy.ID)
 })
 it('isolates training work and refuses to sync it to a live session',async()=>{
  const files=storageFixture()
  const training={username:'alex',name:'Alex',role:'field',duty:'field',mdtUnit:'U1',training:true,exercise:'EX1'}
  files.set('identity',JSON.stringify({session:training}))
  files.set('cache:alex|U1|training:EX1:/mdt',files.get('cache:alex|U1:/mdt')!)
  vi.stubGlobal('fetch',vi.fn().mockRejectedValue(new TypeError('offline')))
  const desktop=await import('./desktop')
  await desktop.desktopRequest('POST','/mdt/status',{status:'oos'})
  expect((await desktop.desktopOutbox())[0].Context).toBe('training:EX1')
  expect(files.get('queue:alex|U1')).toBeUndefined()
  const fetch=vi.fn(async()=>new Response(JSON.stringify({signedIn:true,session:{username:'alex',mdtUnit:'U1'}})))
  vi.stubGlobal('fetch',fetch)
  await desktop.syncDesktop()
  expect(fetch).toHaveBeenCalledTimes(1)
  expect(await desktop.desktopOutbox()).toHaveLength(1)
 })
 it('keeps an own-unit status during an outage and syncs it once connected',async()=>{
  const files=storageFixture();vi.stubGlobal('fetch',vi.fn().mockRejectedValue(new TypeError('offline')))
  const desktop=await import('./desktop')
  const queued=await desktop.desktopRequest('POST','/mdt/status',{status:'oos'})
  expect((await queued.json()).queued).toBe(true)
  const pending=await desktop.desktopOutbox();expect(pending).toHaveLength(1)
  expect(pending[0]).toMatchObject({Owner:'alex',Unit:'U1',Body:{status:'oos'},BaseStatusAt:'2026-11-15T12:00:00Z'})
  const cached=await desktop.desktopRequest('GET','/mdt')
  expect((await cached.json()).unit.status).toBe('oos')
  expect(files.get('queue:alex|U1')).toContain(pending[0].ID)
  vi.stubGlobal('fetch',vi.fn(async(url:string)=>{
   if(url.endsWith('/auth/me'))return new Response(JSON.stringify({signedIn:true,session:{username:'alex',mdtUnit:'U1'}}))
   if(url.endsWith('/offline/sync'))return new Response(null,{status:204})
   return new Response(JSON.stringify({unit:{id:'U1',status:'oos',statusAt:new Date().toISOString()},pending:[],settings:{},health:{}}))
  }))
  await desktop.syncDesktop()
  expect(await desktop.desktopOutbox()).toHaveLength(0)
 })
 it('keeps conflicts in the outbox and refuses to send another account’s work',async()=>{
  storageFixture();vi.stubGlobal('fetch',vi.fn().mockRejectedValue(new TypeError('offline')))
  const desktop=await import('./desktop')
  await desktop.desktopRequest('POST','/mdt/note',{text:'Synthetic field note'})
  const fetch=vi.fn(async(url:string)=>{
   if(url.endsWith('/auth/me'))return new Response(JSON.stringify({signedIn:true,session:{username:'other',mdtUnit:'U2'}}))
   throw new Error('wrong account must not submit')
  });vi.stubGlobal('fetch',fetch)
  await desktop.syncDesktop();expect(fetch).toHaveBeenCalledTimes(1)
  expect(await desktop.desktopOutbox()).toHaveLength(1)
  vi.stubGlobal('fetch',vi.fn(async(url:string)=>url.endsWith('/auth/me')?new Response(JSON.stringify({signedIn:true,session:{username:'alex',mdtUnit:'U1'}})):new Response(JSON.stringify({message:'Dispatch changed the assignment'}),{status:409})))
  await desktop.syncDesktop()
  expect((await desktop.desktopOutbox())[0].error).toContain('Dispatch changed')
 })
 it('searches downloaded local records, retains caution flags and leaves Enformion offline',async()=>{
  const files=storageFixture()
  files.set('cache:alex|U1:/mdt/offline/records',JSON.stringify({at:Date.now(),data:{at:'2026-11-15T12:00:00Z',records:[{record:'R1',name:'SAMPLE, ALEX',dob:'1980-01-02',person:{aliases:['Test Alias'],ssn:'999-12-3456'},flags:[{id:'F1',title:'Synthetic caution',warning:'Test warning',instruction:'Contact dispatch'}]}]}}))
  vi.stubGlobal('fetch',vi.fn().mockRejectedValue(new TypeError('offline')))
  const desktop=await import('./desktop')
  const result=await(await desktop.desktopRequest('POST','/mdt/records/search',{kind:'person',name:'Test Alias'})).json()
  expect(result.items).toHaveLength(1);expect(result.items[0].person.ssn).toBe('999-12-3456')
  expect(result.items[0].flags[0].id).toBe('F1')
  expect(result.providerMessage).toContain('Enformion searches require a connection')
 })
 it('keeps a report draft and its fields available before a server number exists',async()=>{
  storageFixture();vi.stubGlobal('fetch',vi.fn().mockRejectedValue(new TypeError('offline')))
  const desktop=await import('./desktop')
  const saved=await(await desktop.desktopRequest('POST','/rms/reports',{narrative:'Synthetic offline draft',offenses:[],people:[],property:[],vehicles:[]})).json()
  expect(saved.report.number).toMatch(/^OFFLINE-/)
  const draft=await(await desktop.desktopRequest('GET','/rms/reports/'+saved.report.number)).json()
  expect(draft.report.narrative).toBe('Synthetic offline draft')
 })
})

it('opens downloaded person profiles and their associates while fully offline',async()=>{
 const files=storageFixture();files.set('cache:alex|U1:/mdt/offline/records',JSON.stringify({at:Date.now(),data:{at:'2026-11-21',records:[{record:'P1',person:{name:'Alpha',employer:{name:'Test Company'}},associates:[{record:'P2',name:'Beta',type:'friend'}]},{record:'P2',person:{name:'Beta'},associates:[{record:'P1',name:'Alpha',type:'friend'}]}]}}))
 vi.stubGlobal('fetch',vi.fn().mockRejectedValue(new TypeError('offline')))
 const desktop=await import('./desktop')
 const first=await(await desktop.desktopRequest('POST','/mdt/records/person',{record:'P1',purpose:'patrol'})).json()
 expect(first.person.employer.name).toBe('Test Company');expect(first.associates[0].record).toBe('P2')
 const second=await(await desktop.desktopRequest('POST','/mdt/records/person',{record:first.associates[0].record,purpose:'patrol'})).json()
 expect(second.person.name).toBe('Beta')
 expect((await desktop.desktopRequest('POST','/mdt/records/person',{record:'not-downloaded',purpose:'patrol'})).status).toBe(404)
 expect(await desktop.desktopOutbox()).toHaveLength(0)
})
