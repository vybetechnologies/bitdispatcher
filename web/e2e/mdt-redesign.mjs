// Isolated UI regression: synthetic responses only, no production sign-in or writes.
import {chromium} from '@playwright/test'
import {createServer} from 'node:http'
import {readFile,mkdir} from 'node:fs/promises'
import {resolve,extname} from 'node:path'
import assert from 'node:assert/strict'
const root=resolve('../internal/web/dist'),out=resolve('../docs/screenshots/mdt-redesign')
await mkdir(out,{recursive:true})
const types={'.html':'text/html','.js':'application/javascript','.mjs':'application/javascript','.css':'text/css','.woff2':'font/woff2','.webp':'image/webp'}
const server=createServer(async(req,res)=>{try{const path=decodeURIComponent(new URL(req.url,'http://localhost').pathname);const file=resolve(root,'.'+(path==='/'?'/index.html':path));if(!file.startsWith(root))throw Error();res.setHeader('Content-Type',types[extname(file)]||'application/octet-stream');res.end(await readFile(file))}catch{res.statusCode=404;res.end()}})
await new Promise(r=>server.listen(5179,'127.0.0.1',r))
const browser=await chromium.launch({channel:'chrome',headless:true})
const now=new Date().toISOString()
const session={username:'fixture',name:'Test Officer',role:'field',duty:'field',mdtUnit:'TEST-01',created:now,last:now}
const fixture={serverTime:now,unit:{id:'TEST-01',status:'available',statusAt:now,position:{lat:46.8772,lon:-96.7898,at:now}},partners:[],pending:[],statuses:[],messages:[{id:'fixture-msg',at:now,from:'TEST-01',to:['DISPATCH'],text:'Synthetic layout check'}],bolos:[],settings:{agency:'Lakes Area Security',mapCenter:[46.8772,-96.7898],mapZoom:13,markOuts:[{code:'MEAL',name:'Meal break',minutes:30},{code:'FUEL',name:'Fuel',minutes:15},{code:'RPT',name:'Report writing',minutes:45}]},health:{map:{recorded:false},tiles:false,photos:{available:true,attribution:'Test imagery'}},roster:[{id:'fixture',name:'Test Officer'}]}
const writes=[];const errors=[]
try{
 for(const native of [false,true]){
  const context=await browser.newContext({viewport:{width:1440,height:900}})
  await context.addInitScript(({native})=>{
   if(native){const files=new Map();window.lasDesktop={version:'test',call:async(op,key,value)=>{if(op==='read')return files.get(key)||null;if(op==='write')files.set(key,value);if(op==='delete')files.delete(key);return null}}}
   class TestStream {addEventListener(){}close(){}};window.EventSource=TestStream
  },{native})
  const page=await context.newPage();page.on('pageerror',e=>errors.push(String(e)))
  page.setDefaultTimeout(10000)
  await page.route('**/api/**',async route=>{
   const request=route.request();const path=new URL(request.url()).pathname.slice(4);let data={}
   if(request.method()!=='GET')writes.push({path,body:request.postData()})
   if(path==='/auth/me')data={signedIn:true,session}
   else if(path==='/me/keys')data={profile:{id:'fixture',name:'MDT',bindings:[]},actions:{},tested:[]}
   else if(path==='/auth/modules')data={modules:[{id:'mdt',label:'MDT',path:'/mdt'},{id:'admin',label:'Administration',path:'/admin'}]}
   else if(path==='/mdt')data=fixture
   else if(path==='/roads')data={closures:[]}
   else if(path==='/comms')data={items:[],priorities:[]}
   else if(path==='/weather/alerts')data={features:[],checked:now,scope:'Cass / Clay',stale:false}
   else if(path==='/rms/reports')data=[{number:'R-TEST-1',status:'draft',author:'fixture',updated:now,location:'Synthetic training location',offenses:[],people:[],property:[],vehicles:[]}]
   else if(path==='/mdt/records/search'){const q=JSON.parse(request.postData()||'{}');data={kind:q.kind,items:q.kind==='person'?[{record:'P-TEST',seen:now,name:'Test Person',person:{registeredSexOffender:true,offenderProfileUrl:'https://registry.example/fixture',warrants:[{number:'W-TEST',agency:'Test court',offense:'Fixture charge',status:'active'}],cautions:[{type:'Weapons',level:'danger',note:'Fixture caution'}]}}]:[{record:'V-TEST',seen:now,plate:'TEST123',registeredOwner:{record:'P-OWNER',name:'Test Registered Owner',dob:'1980-01-01',person:{id:'P-OWNER',role:'subject',name:'Test Registered Owner',registeredSexOffender:true,offenderProfileUrl:'https://registry.example/owner'}},ownership:'employee',ownerName:'Test Employee',cautions:[{type:'Stolen plate',level:'danger'}]}],source:'Test local records',searchedAt:now}}
   else if(path==='/mdt/offline/records')data={records:[],at:now}
   else if(path==='/mdt/windows-release')data={Available:true,Version:'test'}
   else if(path.startsWith('/map/')||path==='/mdt/offline/map-package'){await route.fulfill({status:404,body:''});return}
   await route.fulfill({json:data})
  })
  await page.goto('http://127.0.0.1:5179/#/mdt')
  try{await page.getByRole('heading',{name:'Your unit',exact:true}).waitFor()}catch(e){await page.screenshot({path:out+'/failure.png'});console.log(JSON.stringify({errors,body:await page.locator('body').innerText()}));throw e}
  const emergency=page.getByRole('button',{name:'Emergency: hold for two seconds'})
  for(const size of [{width:1440,height:900},{width:1366,height:768},{width:1024,height:768},{width:800,height:600},{width:800,height:440},{width:390,height:844}]){
   await page.setViewportSize(size);await page.waitForTimeout(180)
   const box=await emergency.boundingBox();assert(box&&box.y>=0&&box.y+box.height<=size.height+1,`Emergency offscreen: ${native} ${JSON.stringify(size)}`)
   assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),`Horizontal page overflow: ${JSON.stringify(size)}`)
   if(size.width===1440||size.width===800)await page.screenshot({path:`${out}/${native?'windows':'web'}-${size.width}.png`})
  }
  await page.setViewportSize({width:1440,height:900})
  assert.equal(await page.getByLabel('Switch module').count(),native?0:1)
  await page.getByRole('button',{name:'My reports',exact:true}).click()
  await page.getByRole('dialog',{name:'My reports'}).waitFor()
  assert(await emergency.isVisible());await page.screenshot({path:`${out}/${native?'windows':'web'}-reports.png`})
  const beforeEmergency=writes.length
  const emergencyBox=await emergency.boundingBox()
  await page.mouse.move(emergencyBox.x+emergencyBox.width/2,emergencyBox.y+emergencyBox.height/2)
  await page.mouse.down();await page.waitForTimeout(2200);await page.mouse.up();await page.waitForTimeout(250)
  assert(writes.slice(beforeEmergency).some(w=>w.path==='/mdt/emergency'||w.path==='/mdt/offline/sync'&&JSON.parse(w.body).Path==='/mdt/emergency'),'Reports drawer prevented the emergency hold')
  if(await page.getByRole('button',{name:'Close reports'}).count())await page.getByRole('button',{name:'Close reports'}).click()
  await page.getByRole('button',{name:'Voice settings',exact:true}).click()
  await page.getByRole('dialog',{name:'Voice settings'}).waitFor();await page.keyboard.press('Escape')
  await page.getByRole('button',{name:'Persons & vehicles',exact:true}).click()
  await page.getByRole('dialog').waitFor()
  await page.getByLabel('Name or alias').fill('Test Person')
  await page.getByRole('button',{name:'Search LAS records'}).click()
  await page.getByRole('link',{name:'Official offender profile'}).waitFor()
  assert(await page.getByText('Warrant W-TEST · ACTIVE').count()===1)
  await page.getByRole('tab',{name:'Vehicles',exact:true}).click()
  await page.getByLabel('Plate',{exact:true}).fill('TEST123')
  await page.getByRole('button',{name:'Search LAS records'}).click()
  await page.getByText('LAS employee-owned vehicle · Test Employee').waitFor()
  await page.getByText('Stolen plate',{exact:true}).waitFor()
  await page.getByText('Registered owner · Test Registered Owner',{exact:true}).click()
  await page.getByText('Linked local person record P-OWNER',{exact:false}).waitFor()
  assert(await page.getByRole('link',{name:'Official offender profile'}).count()===1)
  await page.keyboard.press('Escape')
  await page.getByRole('button',{name:/Mark out…/}).click();await page.getByLabel('Mark-out note').fill('Synthetic note')
  await page.screenshot({path:`${out}/${native?'windows':'web'}-markout.png`})
  fixture.unit.status='dispatched';fixture.unit.incident='TEST-CALL'
  fixture.typeName='Alarm response';fixture.summary='Synthetic assignment summary for layout verification.'
  fixture.cautions=[{id:'test-caution',name:'Officer safety',note:'Synthetic caution remains visible'}]
  fixture.incident={number:'TEST-CALL',type:'alarm',priority:2,created:now,location:{address:'Training location, Fargo',lat:46.879,lon:-96.79},units:[{unit:'TEST-01',status:'dispatched'}],timeline:[{seq:1,kind:'note',at:now,text:'Synthetic dispatch note',actor:'fixture'}],persons:[{id:'test-person',name:'Test Person',role:'involved',safety:true,registeredSexOffender:true,offenderProfileUrl:'https://registry.example/fixture',warrants:[{number:'W-TEST',agency:'Test court',offense:'Fixture charge',status:'active'}],cautions:[{type:'Weapons',level:'danger',note:'Fixture caution'}]}],vehicles:[{id:'test-vehicle',plate:'TEST123',state:'ND',role:'involved',ownership:'agency',cautions:[{type:'Stolen plate',level:'caution'}]}],files:[]}
  await page.reload();await page.getByText('Synthetic caution remains visible',{exact:false}).waitFor()
  assert(await page.getByRole('button',{name:/En route/}).isEnabled())
  assert(await page.getByRole('link',{name:'Official offender profile'}).count()===1)
  assert(await page.getByText('Warrant W-TEST · ACTIVE').count()===1)
  assert(await page.getByText('Lakes Area Security vehicle',{exact:true}).count()===1)
  await page.getByText('Synthetic dispatch note',{exact:false}).scrollIntoViewIfNeeded()
  assert(await emergency.isVisible())
  await page.screenshot({path:`${out}/${native?'windows':'web'}-assigned.png`})
  await page.getByRole('button',{name:'Zoom in',exact:true}).click()
  await page.getByRole('button',{name:'Zoom out',exact:true}).click()
  await page.getByRole('button',{name:/NWS 0 alerts/}).click()
  await page.getByText('Show affected areas',{exact:false}).waitFor()
  delete fixture.incident;delete fixture.unit.incident;delete fixture.summary;delete fixture.cautions;fixture.unit.status='available'
  assert.equal(errors.length,0,errors.join('\n'))
  await context.close()
 }
 console.log(JSON.stringify({passed:true,modes:['web','windows'],viewports:6,screenshots:out,errors,writes:writes.length}))
}finally{await browser.close();await new Promise(r=>server.close(r))}
