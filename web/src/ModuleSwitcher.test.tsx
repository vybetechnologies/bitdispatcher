import { afterEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, useLocation } from 'react-router-dom'
import { api, ApiError } from './api'
import ModuleSwitcher from './ModuleSwitcher'

const fixture=vi.hoisted(()=>({me:{username:'owner',name:'Owner',role:'field',duty:'field',mdtUnit:'SEC1'}}))
vi.mock('./store',()=>({useStore:(selector:(s:typeof fixture)=>unknown)=>selector(fixture)}))
vi.mock('./offline',()=>({pauseSnapshots:vi.fn(),resumeSnapshots:vi.fn(),clearSnapshot:vi.fn().mockResolvedValue(undefined)}))
vi.mock('sonner',()=>({toast:{error:vi.fn()}}))
afterEach(()=>{cleanup();vi.restoreAllMocks();localStorage.clear()})
function Route(){return <p data-testid="route">{useLocation().pathname}</p>}
const list=[{id:'mdt',label:'MDT',path:'/mdt'},{id:'rms',label:'RMS / Records',path:'/rms'},{id:'admin',label:'Administration',path:'/admin'}]
describe('shared module switcher',()=>{
 it('does not render a toggle for one granted module',async()=>{
  const get=vi.spyOn(api,'get').mockResolvedValue({modules:[list[0]]})
  render(<MemoryRouter initialEntries={['/mdt']}><ModuleSwitcher /></MemoryRouter>)
  await waitFor(()=>expect(get).toHaveBeenCalled())
  expect(screen.queryByLabelText('Switch module')).toBeNull()
 })
 it('shows the granted modules and labels field fallback routes as MDT',async()=>{
  vi.spyOn(api,'get').mockResolvedValue({modules:list})
  render(<MemoryRouter initialEntries={['/console']}><ModuleSwitcher /></MemoryRouter>)
  await waitFor(()=>expect(screen.getByLabelText('Switch module').textContent).toContain('MDT'))
  fireEvent.click(screen.getByLabelText('Switch module'))
  expect(screen.getByText('Administration')).toBeTruthy()
  expect(screen.queryByText('CAD')).toBeNull()
 })
 it('uses the server-checked module id and retains same-duty navigation',async()=>{
  vi.spyOn(api,'get').mockResolvedValue({modules:list})
  const post=vi.spyOn(api,'post').mockResolvedValue({path:'/rms',changed:false,session:fixture.me})
  render(<MemoryRouter initialEntries={['/mdt']}><ModuleSwitcher /><Route /></MemoryRouter>)
  await waitFor(()=>expect(screen.getByLabelText('Switch module')).toBeTruthy())
  fireEvent.click(screen.getByLabelText('Switch module'));fireEvent.click(screen.getByRole('button',{name:'RMS / Records'}))
  await waitFor(()=>expect(screen.getByTestId('route').textContent).toBe('/rms'))
  expect(post).toHaveBeenCalledWith('/auth/module',{module:'rms'})
 })
 it('does not navigate when the server refuses a stale grant',async()=>{
  vi.spyOn(api,'get').mockResolvedValue({modules:list})
  vi.spyOn(api,'post').mockRejectedValue(new ApiError(403,'Access was revoked'))
  render(<MemoryRouter initialEntries={['/mdt']}><ModuleSwitcher /><Route /></MemoryRouter>)
  await waitFor(()=>expect(screen.getByLabelText('Switch module')).toBeTruthy())
  fireEvent.click(screen.getByLabelText('Switch module'));fireEvent.click(screen.getByRole('button',{name:'Administration'}))
  await waitFor(()=>expect(api.post).toHaveBeenCalled())
  expect(screen.getByTestId('route').textContent).toBe('/mdt')
 })
})
