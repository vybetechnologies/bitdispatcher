import {useState} from 'react'
import {afterEach,expect,it,vi} from 'vitest'
import {cleanup,fireEvent,render,screen} from '@testing-library/react'
import PersonRelationships from './PersonRelationships'
import type {PersonRelationship} from './PersonRelationships'
import {api} from '../api'
afterEach(()=>{cleanup();vi.restoreAllMocks()})
it('links a local person, edits the relationship type and removes the association',async()=>{
 vi.spyOn(api,'get').mockResolvedValue({items:[{id:'P2',name:'Test Relative'}]})
 function Editor(){const [value,setValue]=useState<PersonRelationship[]>([]);return <><PersonRelationships value={value} onChange={setValue}/><output>{JSON.stringify(value)}</output></>}
 render(<Editor/>);fireEvent.change(screen.getByLabelText('Find related person'),{target:{value:'Test Relative'}});fireEvent.click(screen.getByRole('button',{name:'Find local person'}));fireEvent.click(await screen.findByRole('button',{name:'Add person relationship'}))
 fireEvent.change(screen.getByLabelText('Relationship',{exact:true}),{target:{value:'parent'}})
 expect(screen.getByRole('status').textContent).toContain('"personId":"P2"');expect(screen.getByRole('status').textContent).toContain('"type":"parent"')
 fireEvent.click(screen.getByRole('button',{name:'Remove relationship'}));expect(screen.getByRole('status').textContent).toBe('[]')
})
