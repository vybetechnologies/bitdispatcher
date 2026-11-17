import {afterEach,expect,it,vi} from 'vitest'
import {cleanup,fireEvent,render,screen,waitFor} from '@testing-library/react'
import IncidentActions from './IncidentActions'
import {api} from '../api'
import type {MDTView} from './Mdt'
vi.mock('../ui',()=>({Modal:({open,children,title}:{open:boolean;children:React.ReactNode;title:string})=>open?<section role="dialog" aria-label={title}>{children}</section>:null}))
afterEach(()=>{cleanup();vi.restoreAllMocks()})
const fixture={unit:{id:'TEST1',status:'available'},callTypes:[{code:'PATROL',name:'Patrol',priority:3}],dispositions:['Resolved']} as MDTView
it('creates an officer call with the chosen type and location',async()=>{
 const saved=vi.fn(),post=vi.spyOn(api,'post').mockResolvedValue({number:'TEST-CALL'})
 render(<IncidentActions m={fixture} onSaved={saved}/>)
 fireEvent.click(screen.getByRole('button',{name:'New incident'}))
 fireEvent.change(screen.getByLabelText('Call type'),{target:{value:'PATROL'}})
 fireEvent.change(screen.getByLabelText('Location'),{target:{value:'Synthetic location'}})
 fireEvent.click(screen.getByRole('button',{name:'Create and assign to me'}))
 await waitFor(()=>expect(saved).toHaveBeenCalled())
 expect(post).toHaveBeenCalledWith('/mdt/incidents',expect.objectContaining({type:'PATROL',location:{address:'Synthetic location'}}))
})
it('requires closing the currently assigned incident using a disposition',async()=>{
 const saved=vi.fn(),post=vi.spyOn(api,'post').mockResolvedValue(undefined)
 render(<IncidentActions m={{...fixture,unit:{...fixture.unit,incident:'TEST-CALL',status:'onscene'},incident:{number:'TEST-CALL'} as NonNullable<MDTView['incident']>}} onSaved={saved}/>)
 expect(screen.getByRole('button',{name:'New incident'}).hasAttribute('disabled')).toBe(true)
 fireEvent.click(screen.getByRole('button',{name:'Close incident'}))
 fireEvent.change(screen.getByLabelText('Disposition'),{target:{value:'Resolved'}})
 fireEvent.click(screen.getAllByRole('button',{name:'Close incident'})[1])
 await waitFor(()=>expect(saved).toHaveBeenCalled())
 expect(post).toHaveBeenCalledWith('/mdt/close',{incident:'TEST-CALL',disposition:'Resolved',text:''})
})
