import { afterEach, expect, it, vi } from 'vitest'
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import MdtVoice, { changedAnnouncements, voiceSnapshot } from './Voice'
import type { MDTView } from './Mdt'
import type { CommView } from '../types'
import type { WeatherFeature } from '../WeatherAlerts'
const m={unit:{id:'SEC1',status:'dispatched'},incident:{number:'LAS123',type:'ALARM',priority:2,status:'open',location:{address:'100 Main St'},timeline:[]},typeName:'Alarm',cautions:[{id:'dog',name:'Dog',note:'Loose dog'}],bolos:[{id:'b1',text:'Blue vehicle',plate:'ABC'}]} as unknown as MDTView
afterEach(()=>{cleanup();vi.unstubAllGlobals()})
const weather:WeatherFeature={type:'Feature',id:'wx1',geometry:null,properties:{id:'wx1',event:'Blizzard Warning',areaDesc:'Cass and Clay',headline:'Whiteout conditions',instruction:'Avoid travel',description:'Snow and wind',severity:'Severe',urgency:'Immediate',certainty:'Observed',sent:new Date().toISOString(),effective:new Date().toISOString(),expires:new Date(Date.now()+3600000).toISOString()}}
function speechMock(){
 const utterances: {text:string;onend?:()=>void}[]=[]
 const speak=vi.fn((u:{text:string;onend?:()=>void})=>utterances.push(u))
 const cancel=vi.fn()
 vi.stubGlobal('speechSynthesis',{getVoices:()=>[{name:'Local',voiceURI:'local',lang:'en-US',localService:true}],speak,cancel,addEventListener:vi.fn(),removeEventListener:vi.fn()})
 vi.stubGlobal('SpeechSynthesisUtterance',class{constructor(public text:string){}})
 const drain=()=>{let n=0;while(utterances.length && n++<30)utterances.shift()!.onend?.()}
 return {speak,cancel,drain}
}
it('speaks a changed unit status once after receiving the confirmed terminal state',async()=>{
 const audio=speechMock()
 const base={...m,cautions:[],bolos:[],incident:undefined}
 const panel=render(<MdtVoice m={base} messages={[]} connected/> )

 await waitFor(()=>expect(screen.getByText('Voice on')).toBeTruthy())
 audio.drain();audio.speak.mockClear()
 const changed={...base,unit:{...base.unit,status:'enroute'}}
 panel.rerender(<MdtVoice m={changed} messages={[]} connected/> )
 await waitFor(()=>expect(audio.speak).toHaveBeenCalledTimes(1))
 expect(audio.speak.mock.calls[0][0].text).toContain('En route')
 audio.drain()
 panel.rerender(<MdtVoice m={{...changed}} messages={[]} connected/> )
 expect(audio.speak).toHaveBeenCalledTimes(1)
})
it('includes routine bulletins and other-unit messages, excluding the officer’s own messages',()=>{
 const next=voiceSnapshot({...m,messages:[{id:'peer',from:'SEC2',fromUnit:true,text:'Meet at gate',to:['SEC1'],at:new Date().toISOString()},{id:'self',from:'SEC1',fromUnit:true,text:'Copy',to:['SEC2'],at:new Date().toISOString()}]},[{id:'routine',kind:'message',priority:'low',state:'active',subject:'Shift update',body:'North gate closed',mine:false} as CommView])
 expect(next.find(n=>n.id==='message:peer')?.text).toContain('Meet at gate')
 expect(next.find(n=>n.id==='comm:routine')?.text).toContain('North gate closed')
 expect(next.some(n=>n.id==='message:self')).toBe(false)
})
it('announces new and changed weather instructions once and does not replay on recovery',async()=>{
 const audio=speechMock()
 const base={...m,cautions:[],bolos:[],incident:undefined}
 const panel=render(<MdtVoice m={base} messages={[]} connected/> )

 await waitFor(()=>expect(screen.getByText('Voice on')).toBeTruthy())
 audio.drain();audio.speak.mockClear()
 panel.rerender(<MdtVoice m={base} messages={[]} weather={[weather]} connected/> )
 expect(audio.speak.mock.calls[0][0].text).toContain('Blizzard Warning for Cass and Clay')
 audio.drain()
 panel.rerender(<MdtVoice m={base} messages={[]} weather={[]} connected/> )
 panel.rerender(<MdtVoice m={base} messages={[]} weather={[weather]} connected/> )
 expect(audio.speak).toHaveBeenCalledTimes(1)
 panel.rerender(<MdtVoice m={base} messages={[]} weather={[{...weather,properties:{...weather.properties,instruction:'Shelter indoors'}}]} connected/> )
 expect(audio.speak.mock.calls[1][0].text).toContain('Shelter indoors')
})
it('prioritizes safety and deduplicates polling while announcing changed status',()=>{
 const snap=voiceSnapshot(m,[])
 expect(snap[0].text).toContain('Loose dog')
 const seen=new Map(snap.map(s=>[s.id,s.text]))
 expect(changedAnnouncements(seen,voiceSnapshot(m,[]))).toHaveLength(0)
 const next=voiceSnapshot({...m,unit:{...m.unit,status:'enroute'}},[])
 expect(changedAnnouncements(seen,next).map(n=>n.id)).toEqual(['status'])
})
it('keeps the default-on intent visible when the browser requires a gesture',async()=>{
 const audio=speechMock()
 render(<MdtVoice m={m} messages={[]} connected/> )
 await waitFor(()=>expect(audio.speak).toHaveBeenCalled())
 const utterance=audio.speak.mock.calls[0][0] as unknown as {onerror:(e:{error:string})=>void}
 act(()=>utterance.onerror({error:'not-allowed'}))
 const button=screen.getByText('Voice on · Enable audio')
 expect(button.getAttribute('aria-pressed')).toBe('true')
 fireEvent.click(button)
 await waitFor(()=>expect(screen.getByText('Voice on')).toBeTruthy())
})
it('starts automatically with local voices and cancels on lock',async()=>{
 const speak=vi.fn();const cancel=vi.fn()
 vi.stubGlobal('speechSynthesis',{getVoices:()=>[{name:'Local',voiceURI:'local',lang:'en-US',localService:true},{name:'Cloud',voiceURI:'cloud',lang:'en-US',localService:false}],speak,cancel,addEventListener:vi.fn(),removeEventListener:vi.fn()})
 vi.stubGlobal('SpeechSynthesisUtterance',class {constructor(public text:string){}})
 render(<MdtVoice m={m} messages={[]} connected/> )

 fireEvent.click(screen.getByText('Voice settings'))
 expect(screen.queryByText('Cloud')).toBeNull()

 await waitFor(()=>expect(speak).toHaveBeenCalled())
 expect(speak.mock.calls[0][0].voice.voiceURI).toBe('local')
 window.dispatchEvent(new Event('bd:locked'))
 await waitFor(()=>expect(screen.getByText('Voice off')).toBeTruthy())
 expect(cancel).toHaveBeenCalled()
})
