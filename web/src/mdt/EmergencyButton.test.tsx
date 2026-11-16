import { afterEach, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import EmergencyButton from './EmergencyButton'
import { api } from '../api'
vi.mock('sonner',()=>({toast:{error:vi.fn()}}))
afterEach(()=>{cleanup();vi.useRealTimers();vi.restoreAllMocks()})
it('cancels a short hold, sends one emergency after two seconds and supports keyboard',async()=>{
 vi.useFakeTimers();const post=vi.spyOn(api,'post').mockResolvedValue(undefined)
 render(<EmergencyButton unitParam="" onSent={()=>{}}/> )
 const button=screen.getByRole('button',{name:'Emergency: hold for two seconds'})
 fireEvent.keyDown(button,{key:' '});vi.advanceTimersByTime(1000);fireEvent.keyUp(button,{key:' '});vi.advanceTimersByTime(2000)
 expect(post).not.toHaveBeenCalled()
 fireEvent.keyDown(button,{key:'Enter'});fireEvent.keyDown(button,{key:'Enter',repeat:true});vi.advanceTimersByTime(2100)
 expect(post).toHaveBeenCalledTimes(1);expect(post).toHaveBeenCalledWith('/mdt/emergency')
})
it('clears an unfinished hold when the terminal is removed',()=>{
 vi.useFakeTimers();const post=vi.spyOn(api,'post').mockResolvedValue(undefined)
 const page=render(<EmergencyButton unitParam="" onSent={()=>{}}/> )
 fireEvent.keyDown(screen.getByRole('button'),{key:'Enter'});page.unmount();vi.advanceTimersByTime(3000)
 expect(post).not.toHaveBeenCalled()
})
