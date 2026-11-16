import { afterEach, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import LasRecords from './LasRecords'
import { api } from '../api'
vi.mock('./Admin',()=>({Page:({children,title}:{children:React.ReactNode;title:string})=><div><h1>{title}</h1>{children}</div>,Table:({children}:{children:React.ReactNode})=><table><tbody>{children}</tbody></table>}))
vi.mock('sonner',()=>({toast:{error:vi.fn(),success:vi.fn()}}))
afterEach(()=>{cleanup();vi.restoreAllMocks()})
it('creates an officer-searchable person using the administration API',async()=>{
 vi.spyOn(api,'get').mockResolvedValue({items:[],total:0});const post=vi.spyOn(api,'post').mockResolvedValue({id:'LASR-1'})
 render(<LasRecords/>);fireEvent.click(screen.getByText('Create record'))
 fireEvent.change(screen.getByLabelText('Full name'),{target:{value:'Test Person'}})
 fireEvent.click(screen.getByText('Save record'))
 await waitFor(()=>expect(post).toHaveBeenCalledWith('/admin/las-records',expect.objectContaining({kind:'person',name:'Test Person',source:'LAS administrative entry'})))
})
it('requires validated preview before committing an import',async()=>{
 vi.spyOn(api,'get').mockResolvedValue({items:[],total:0})
 const post=vi.spyOn(api,'post').mockResolvedValue({create:1,skipped:0,total:1,preview:[{kind:'vehicle',plate:'TEST12',source:'Fixture'}]})
 render(<LasRecords/> )
 expect(screen.queryByText('Import 1 records')).toBeNull()
 fireEvent.change(screen.getByLabelText('Records import file'),{target:{files:[{name:'fixture.csv',size:70,text:async()=> 'kind,plate,source\nvehicle,TEST12,Fixture\n'}]}})
 await waitFor(()=>expect((screen.getByText('Validate and preview') as HTMLButtonElement).disabled).toBe(false))
 fireEvent.click(screen.getByText('Validate and preview'))
 await waitFor(()=>expect(screen.getByText('Import 1 records')).toBeTruthy())
 expect(post).toHaveBeenCalledWith('/admin/las-records/import',expect.objectContaining({format:'csv',commit:false}))
 post.mockResolvedValueOnce({created:1,skipped:0})
 fireEvent.click(screen.getByText('Import 1 records'))
 await waitFor(()=>expect(post).toHaveBeenCalledWith('/admin/las-records/import',expect.objectContaining({commit:true})))
})
