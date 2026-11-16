import { afterEach, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import RecordPersonPicker from './RecordPersonPicker'
import { api } from '../api'
afterEach(()=>{cleanup();vi.restoreAllMocks()})
it('links full record information while keeping this incident’s role and safety context',async()=>{
 const use=vi.fn()
 vi.spyOn(api,'post').mockResolvedValue({items:[{id:'LASR-1',name:'PERSON, TEST',dob:'1980-01-01',person:{id:'',role:'subject',first:'Test',last:'Person',ssn:'123-45-6789',dl:'FULL-DL',address:'Test address',safety:true,safetyNote:{title:'Other incident',scope:'incident'}},incidents:[{incident:'OLD-1',role:'witness'}]}]})
 render(<RecordPersonPicker value={{id:'draft-1',role:'caller',roles:['caller'],locKind:'incident',clothing:'Current coat',safety:false}} onUse={use}/> )
 fireEvent.change(screen.getByLabelText('Find person record'),{target:{value:'Test Person'}})
 fireEvent.click(screen.getByText('Find record'))
 await waitFor(()=>expect(screen.getByText('Use this person')).toBeTruthy())
 fireEvent.click(screen.getByText('Use this person'))
 expect(use).toHaveBeenCalledWith(expect.objectContaining({recordId:'LASR-1',id:'draft-1',role:'caller',ssn:'123-45-6789',dl:'FULL-DL',clothing:'Current coat',safety:false,safetyNote:undefined}))
})
