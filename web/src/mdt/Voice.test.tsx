import { afterEach, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import MdtVoice, { changedAnnouncements, voiceSnapshot } from './Voice'
import type { MDTView } from './Mdt'
const m={unit:{id:'SEC1',status:'dispatched'},incident:{number:'LAS123',type:'ALARM',priority:2,status:'open',location:{address:'100 Main St'},timeline:[]},typeName:'Alarm',cautions:[{id:'dog',name:'Dog',note:'Loose dog'}],bolos:[{id:'b1',text:'Blue vehicle',plate:'ABC'}]} as unknown as MDTView
afterEach(()=>{cleanup();vi.unstubAllGlobals()})
it('prioritizes safety and deduplicates polling while announcing changed status',()=>{
 const snap=voiceSnapshot(m,[])
 expect(snap[0].text).toContain('Loose dog')
 const seen=new Map(snap.map(s=>[s.id,s.text]))
 expect(changedAnnouncements(seen,voiceSnapshot(m,[]))).toHaveLength(0)
 const next=voiceSnapshot({...m,unit:{...m.unit,status:'enroute'}},[])
 expect(changedAnnouncements(seen,next).map(n=>n.id)).toEqual(['status'])
})
it('uses only local voices, requires enabling, and cancels on mute and lock',async()=>{
 const speak=vi.fn();const cancel=vi.fn()
 vi.stubGlobal('speechSynthesis',{getVoices:()=>[{name:'Local',voiceURI:'local',lang:'en-US',localService:true},{name:'Cloud',voiceURI:'cloud',lang:'en-US',localService:false}],speak,cancel,addEventListener:vi.fn(),removeEventListener:vi.fn()})
 vi.stubGlobal('SpeechSynthesisUtterance',class {constructor(public text:string){}})
 render(<MdtVoice m={m} messages={[]} connected/> )
 expect(speak).not.toHaveBeenCalled()
 fireEvent.click(screen.getByText('Voice settings'))
 expect(screen.queryByText('Cloud')).toBeNull()
 fireEvent.click(screen.getByText('Voice off'))
 await waitFor(()=>expect(speak).toHaveBeenCalled())
 expect(speak.mock.calls[0][0].voice.voiceURI).toBe('local')
 window.dispatchEvent(new Event('bd:locked'))
 await waitFor(()=>expect(screen.getByText('Voice off')).toBeTruthy())
 expect(cancel).toHaveBeenCalled()
})
