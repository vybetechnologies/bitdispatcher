import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import WeatherAlerts, { liveWeather, weatherColor, type WeatherFeed } from './WeatherAlerts'
import { api } from './api'
afterEach(()=>{cleanup();vi.restoreAllMocks()})
const now=Date.now()
const feed:WeatherFeed={type:'FeatureCollection',checked:new Date(now).toISOString(),scope:'Cass / Clay',stale:false,features:[{type:'Feature',id:'ice',geometry:null,properties:{id:'ice',event:'Ice Storm Warning',headline:'Dangerous icing',description:'Ice accumulation',instruction:'Avoid travel',severity:'Severe',certainty:'Likely',urgency:'Expected',areaDesc:'Cass',sent:new Date(now).toISOString(),effective:new Date(now).toISOString(),expires:new Date(now+60000).toISOString()}}]}
it('removes expired warnings and data older than 30 minutes',()=>{
 expect(liveWeather(feed,now)).toHaveLength(1)
 expect(liveWeather(feed,now+61000)).toHaveLength(0)
 expect(liveWeather({...feed,checked:new Date(now-31*60000).toISOString()},now)).toHaveLength(0)
 expect(weatherColor('Blizzard Warning')).toBe('#f43f5e')
})
it('keeps a boundaryless warning visible and shows NWS instructions',async()=>{
 vi.spyOn(api,'get').mockResolvedValue(feed)
 render(<WeatherAlerts map={{current:null}} ready={false}/> )
 await waitFor(()=>expect(screen.getByText('NWS 1 alert')).toBeTruthy())
 fireEvent.click(screen.getByText('NWS 1 alert'))
 fireEvent.click(screen.getByText('Ice Storm Warning · Severe'))
 expect(screen.getByText('Avoid travel')).toBeTruthy()
 expect(screen.getByText(/Boundary unavailable/)).toBeTruthy()
})
it('does not claim no alerts during an outage',async()=>{
 vi.spyOn(api,'get').mockRejectedValue(new Error('offline'))
 render(<WeatherAlerts map={{current:null}} ready={false}/>)
 await waitFor(()=>expect(screen.getByText('NWS unavailable / stale')).toBeTruthy())
 fireEvent.click(screen.getByText('NWS unavailable / stale'))
 expect(screen.queryByText(/No active NWS/)).toBeNull()
 expect(screen.getByText(/Weather coverage is not current/)).toBeTruthy()
})
it('shares only fresh active weather with the MDT voice reader',async()=>{
 const spoken=vi.fn()
 vi.spyOn(api,'get').mockResolvedValue({...feed,stale:true})
 render(<WeatherAlerts map={{current:null}} ready={false} onAlerts={spoken}/>)
 await waitFor(()=>expect(screen.getByText('NWS unavailable / stale')).toBeTruthy())
 expect(spoken.mock.calls.every(([alerts])=>alerts.length===0)).toBe(true)
})
