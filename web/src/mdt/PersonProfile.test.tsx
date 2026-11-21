import {afterEach,expect,it,vi} from 'vitest'
import {cleanup,fireEvent,render,screen,waitFor} from '@testing-library/react'
import PersonProfile from './PersonProfile'
import {api} from '../api'
afterEach(()=>{cleanup();vi.restoreAllMocks()})
it('follows associates across multiple profiles and handles a cycle with a bounded path',async()=>{
 const graph:Record<string,unknown>={A:{record:'A',name:'Alpha',seen:'',person:{employer:{name:'Test Company'}},associates:[{record:'B',name:'Beta',type:'parent'}]},B:{record:'B',name:'Beta',seen:'',associates:[{record:'C',name:'Gamma',type:'friend'}]},C:{record:'C',name:'Gamma',seen:'',associates:[{record:'A',name:'Alpha',type:'associate'}]}}
 const post=vi.spyOn(api,'post').mockImplementation(async(_path,body)=>graph[(body as {record:string}).record] as never),loaded=vi.fn()
 render(<PersonProfile initial="A" purpose="patrol" onBack={()=>{}} onLoaded={loaded} renderPerson={()=>null}/>)
 await screen.findByRole('heading',{name:'Alpha'});expect(screen.getByText('Employer · Test Company')).toBeTruthy()
 fireEvent.click(screen.getByRole('button',{name:'parent · Beta'}));await screen.findByRole('heading',{name:'Beta'})
 fireEvent.click(screen.getByRole('button',{name:'friend · Gamma'}));await screen.findByRole('heading',{name:'Gamma'})
 fireEvent.click(screen.getByRole('button',{name:'associate · Alpha'}));await screen.findByRole('heading',{name:'Alpha'})
 expect(screen.queryByRole('button',{name:'Previous person'})).toBeNull();expect(loaded).toHaveBeenCalledTimes(4)
 expect(post.mock.calls.every(([path])=>path==='/mdt/records/person')).toBe(true)
})
it('shows an unavailable associate without exposing stale previous profile details',async()=>{
 vi.spyOn(api,'post').mockResolvedValueOnce({record:'A',name:'Alpha',seen:'',associates:[{record:'B',name:'Beta',type:'friend'}]}).mockRejectedValueOnce(Error('Person is unavailable'))
 render(<PersonProfile initial="A" purpose="patrol" onBack={()=>{}} onLoaded={()=>{}} renderPerson={()=>null}/>)
 await screen.findByRole('heading',{name:'Alpha'});fireEvent.click(screen.getByRole('button',{name:'friend · Beta'}))
 await waitFor(()=>expect(screen.getByRole('alert').textContent).toContain('Person is unavailable'))
 expect(screen.queryByRole('heading',{name:'Alpha'})).toBeNull()
})
