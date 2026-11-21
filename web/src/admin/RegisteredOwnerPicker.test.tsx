import {afterEach,expect,it,vi} from 'vitest'
import {cleanup,fireEvent,render,screen,waitFor} from '@testing-library/react'
import RegisteredOwnerPicker from './RegisteredOwnerPicker'
import {api} from '../api'
afterEach(()=>{cleanup();vi.restoreAllMocks()})
it('leaves ownership optional, looks up only local people, and links the selected record ID',async()=>{
 const get=vi.spyOn(api,'get').mockResolvedValue({items:[{id:'LASR-P1',name:'Test Person',dob:'1980-01-01'}]})
 const post=vi.spyOn(api,'post'),change=vi.fn()
 render(<RegisteredOwnerPicker onChange={change}/>)
 expect(screen.getByText('No person record linked.')).toBeTruthy();expect(change).not.toHaveBeenCalled()
 fireEvent.change(screen.getByLabelText('Find registered owner'),{target:{value:'Test Person'}})
 fireEvent.click(screen.getByText('Find local person'))
 await waitFor(()=>expect(screen.getByText('Link as registered owner')).toBeTruthy())
 expect(get).toHaveBeenCalledWith('/admin/las-records?kind=person&q=Test%20Person');expect(post).not.toHaveBeenCalled()
 fireEvent.click(screen.getByText('Link as registered owner'));expect(change).toHaveBeenCalledWith('LASR-P1','Test Person')
})
it('removes the record link independently of vehicle flags and affiliation',()=>{
 const change=vi.fn();render(<RegisteredOwnerPicker value="LASR-P1" name="Test Person" onChange={change}/>)
 fireEvent.click(screen.getByText('Remove registered-owner link'));expect(change).toHaveBeenCalledWith('','')
})
