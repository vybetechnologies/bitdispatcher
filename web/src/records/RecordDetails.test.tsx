import {afterEach,expect,it,vi} from 'vitest'
import {cleanup,fireEvent,render,screen} from '@testing-library/react'
import {activeWarrant,PersonRecordEditor,RecordDetailsDisplay,recordDetailsAnnouncement} from './RecordDetails'
afterEach(()=>{cleanup();vi.useRealTimers()})
it('announces only active unexpired warrants while keeping all warrant history visible',()=>{
 vi.useFakeTimers();vi.setSystemTime(new Date('2026-11-21T12:00:00'))
 const warrants=[{number:'ACTIVE',agency:'Test court',offense:'Test charge',status:'active' as const,expires:'2026-11-21'},{number:'OLD',agency:'Test court',offense:'Old charge',status:'active' as const,expires:'2026-11-20'},{number:'SERVED',agency:'Test court',offense:'Served charge',status:'served' as const}]
 expect(activeWarrant(warrants[0])).toBe(true)
 const text=recordDetailsAnnouncement({warrants,registeredSexOffender:true,cautions:[{type:'Weapons',level:'danger',note:'Contact dispatch'}]})
 expect(text).toContain('ACTIVE');expect(text).not.toContain('OLD');expect(text).not.toContain('SERVED');expect(text).toContain('Registered sex offender');expect(text).toContain('Weapons')
 render(<RecordDetailsDisplay person={{warrants}}/>);expect(screen.getByText(/expired by date/)).toBeTruthy();expect(screen.getByText(/SERVED/)).toBeTruthy()
})
it('keeps vehicle affiliation informational and reads vehicle cautions',()=>{
 render(<RecordDetailsDisplay ownership="employee" ownerName="Test Employee" cautions={[{type:'Stolen plate',level:'danger'}]}/> )
 expect(screen.getByText(/employee-owned/).className).toContain('teal')
 expect(recordDetailsAnnouncement(undefined,[{type:'Stolen plate',level:'danger'}],'agency')).toContain('Lakes Area Security vehicle')
 expect(recordDetailsAnnouncement(undefined,[{type:'Stolen plate',level:'danger'}])).toContain('Stolen plate')
})
it('edits registry status and official HTTPS profile without replacing core safety fields',()=>{
 const change=vi.fn();render(<PersonRecordEditor value={{}} onChange={change}/>)
 fireEvent.click(screen.getByLabelText('Registered sex offender'));expect(change).toHaveBeenCalledWith({registeredSexOffender:true})
 fireEvent.change(screen.getByLabelText('Official offender profile URL'),{target:{value:'https://registry.example/profile/1'}})
 expect(change).toHaveBeenCalledWith({offenderProfileUrl:'https://registry.example/profile/1'})
})
it('opens official links safely and rejects executable link display',()=>{
 const {rerender}=render(<RecordDetailsDisplay person={{registeredSexOffender:true,offenderProfileUrl:'https://registry.example/1'}}/> )
 expect(screen.getByRole('link').getAttribute('rel')).toBe('noopener noreferrer')
 rerender(<RecordDetailsDisplay person={{registeredSexOffender:true,offenderProfileUrl:'javascript:alert(1)'}}/> )
 expect(screen.queryByRole('link')).toBeNull()
})
