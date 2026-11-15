import { afterEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { api } from '../api'
import RecordSearch from './RecordSearch'

vi.mock('../ui',()=>({Modal:({children,title}:{children:React.ReactNode;title:string})=><section aria-label={title}>{children}</section>}))
afterEach(()=>{cleanup();vi.restoreAllMocks()})
const empty={kind:'person',items:[],more:false,source:'LAS CAD records',searchedAt:'2026-11-15T12:00:00Z'}
describe('MDT private record lookup',()=>{
 it('submits a person search only to the local MDT endpoint and explains no matches',async()=>{
  const post=vi.spyOn(api,'post').mockResolvedValue(empty)
  render(<RecordSearch onClose={()=>{}} />)
  fireEvent.change(screen.getByLabelText('Name or alias'),{target:{value:'Example Person'}})
  fireEvent.change(screen.getByLabelText('Date of birth (optional)'),{target:{value:'1982-02-03'}})
  fireEvent.click(screen.getByRole('button',{name:'Search LAS records'}))
  await waitFor(()=>expect(screen.getByRole('status').textContent).toContain('No matching LAS records'))
  expect(post).toHaveBeenCalledWith('/mdt/records/search',{kind:'person',purpose:'assigned-call',name:'Example Person',dob:'1982-02-03'})
  expect(screen.getByText(/No match does not establish/)).toBeTruthy()
 })
 it('switches to vehicles and sends only vehicle criteria',async()=>{
  const post=vi.spyOn(api,'post').mockResolvedValue({...empty,kind:'vehicle'})
  render(<RecordSearch onClose={()=>{}} />)
  fireEvent.click(screen.getByRole('tab',{name:'Vehicles'}))
  fireEvent.change(screen.getByLabelText('Plate'),{target:{value:'abc123'}})
  fireEvent.change(screen.getByLabelText('State (optional)'),{target:{value:'nd'}})
  fireEvent.click(screen.getByRole('button',{name:'Search LAS records'}))
  await waitFor(()=>expect(post).toHaveBeenCalledWith('/mdt/records/search',{kind:'vehicle',purpose:'assigned-call',plate:'ABC123',state:'ND',vin:''}))
 })
 it('shows failures as errors rather than a clear or no-record response',async()=>{
  vi.spyOn(api,'post').mockRejectedValue(new Error('Your MDT assignment has changed'))
  render(<RecordSearch onClose={()=>{}} />)
  fireEvent.change(screen.getByLabelText('Name or alias'),{target:{value:'Example Person'}})
  fireEvent.click(screen.getByRole('button',{name:'Search LAS records'}))
  await waitFor(()=>expect(screen.getByRole('alert').textContent).toContain('assignment has changed'))
  expect(screen.queryByText('No matching LAS records.')).toBeNull()
 })
})
