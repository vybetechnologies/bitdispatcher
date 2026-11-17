import {afterEach,expect,it,vi} from 'vitest'
import {cleanup,render,screen,waitFor} from '@testing-library/react'
import FbiWanted,{fbiAnnouncement} from './FbiWanted'
import {api} from '../api'
afterEach(()=>{cleanup();vi.restoreAllMocks()})
it('speaks and displays a distinct possible FBI match with official caution',async()=>{
 const announce=vi.fn();const notice={uid:'FBI-TEST',title:'TEST PERSON',url:'https://www.fbi.gov/wanted/test',subjects:['Fugitives'],caution:'Published test caution',warning_message:'Published test warning'}
 vi.spyOn(api,'post').mockResolvedValue({items:[notice],checked:'2026-11-17T12:00:00Z',more:false})
 render(<FbiWanted query={{id:1,name:'Test Person',dob:'',purpose:'patrol'}} onCaution={announce}/>)
 await waitFor(()=>expect(announce).toHaveBeenCalledTimes(1))
 expect(announce.mock.calls[0][0]).toContain('Possible FBI Wanted notice match')
 expect(announce.mock.calls[0][0]).toContain('Published test caution')
 expect(announce.mock.calls[0][0]).toContain('not confirmed')
 expect(screen.getByText('FBI caution: Published test caution')).toBeTruthy()
 expect(screen.getByRole('link',{name:'Official FBI notice'}).getAttribute('href')).toBe(notice.url)
})
it('reports FBI failure without emitting a false caution or no-match claim',async()=>{
 const announce=vi.fn();vi.spyOn(api,'post').mockRejectedValue(new Error('FBI unavailable'))
 render(<FbiWanted query={{id:1,name:'Test Person',dob:'',purpose:'patrol'}} onCaution={announce}/>)
 await waitFor(()=>expect(screen.getByText(/FBI lookup unavailable/)).toBeTruthy())
 expect(announce).not.toHaveBeenCalled();expect(fbiAnnouncement([])).toBe('')
})
