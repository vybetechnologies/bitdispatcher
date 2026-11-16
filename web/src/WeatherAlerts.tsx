import { useEffect, useState } from 'react'
import type { Map as GLMap, GeoJSONSource } from 'maplibre-gl'
import { CloudSnow } from 'lucide-react'
import { api } from './api'

export type WeatherProperties = { id: string; event: string; headline: string; description: string; instruction: string; severity: string; certainty: string; urgency: string; areaDesc: string; sent: string; effective: string; onset?: string; ends?: string; expires: string }
export type WeatherFeature = GeoJSON.Feature<GeoJSON.Geometry | null, WeatherProperties> & { id: string }
export type WeatherFeed = { type: 'FeatureCollection'; features: WeatherFeature[]; checked: string; scope: string; stale: boolean; error?: string }
export function liveWeather(feed: WeatherFeed | null, now = Date.now()): WeatherFeature[] {
  if (!feed || !feed.checked || now - Date.parse(feed.checked) > 30 * 60000) return []
  return feed.features.filter(f => Date.parse(f.properties.expires) > now && (!f.properties.ends || Date.parse(f.properties.ends) > now))
}
export function weatherColor(event: string) {
  return /warning/i.test(event) ? '#f43f5e' : /watch/i.test(event) ? '#fbbf24' : '#38bdf8'
}

/** Same public, server-cached feed on desk and vehicle maps; enabled by default. */
export default function WeatherAlerts({ map, ready }: { map: React.RefObject<GLMap | null>; ready: boolean }) {
  const [feed, setFeed] = useState<WeatherFeed | null>(null)
  const [failed, setFailed] = useState(false)
  const [now, setNow] = useState(Date.now())
  const [open, setOpen] = useState(false)
  const [show, setShow] = useState(true)
  const [selected, setSelected] = useState<string>()
  useEffect(() => {
    let alive = true
    const load = () => api.get<WeatherFeed>('/weather/alerts').then(v => { if (alive) { setFeed(v); setFailed(false) } }).catch(() => { if (alive) setFailed(true) })
    load()
    const poll = window.setInterval(load, 60000)
    const tick = window.setInterval(() => setNow(Date.now()), 15000)
    const offline=()=>setFailed(true)
    window.addEventListener('online', load)
    window.addEventListener('offline',offline)
    return () => { alive = false; clearInterval(poll); clearInterval(tick); window.removeEventListener('online', load);window.removeEventListener('offline',offline) }
  }, [])
  const alerts = liveWeather(feed, now).sort((a,b) => ['Extreme','Severe','Moderate','Minor','Unknown'].indexOf(a.properties.severity) - ['Extreme','Severe','Moderate','Minor','Unknown'].indexOf(b.properties.severity))
  const stale = failed || feed?.stale || (!!feed?.checked && now - Date.parse(feed.checked) > 2 * 60000)
  useEffect(() => {
    const m = map.current
    if (!m || !ready) return
    if (!m.getSource('las-weather')) {
      m.addSource('las-weather', { type:'geojson', data:{type:'FeatureCollection',features:[]} })
      // Insert below operational overlays so weather never hides roads/calls/routes.
      const before = m.getStyle().layers?.find(l => ['grid','fence-fill','palert-fill','traffic','route-casing'].includes(l.id))?.id
      m.addLayer({ id:'las-weather-fill',type:'fill',source:'las-weather',paint:{'fill-color':['get','color'],'fill-opacity':0.13} }, before)
      m.addLayer({ id:'las-weather-line',type:'line',source:'las-weather',paint:{'line-color':['get','color'],'line-width':2,'line-opacity':0.85} }, before)
    }
    const click = (e: { features?: GeoJSON.Feature[] }) => {
      const id = String(e.features?.[0]?.properties?.alertID ?? '')
      if (id) { setSelected(id); setOpen(true) }
    }
    m.on('click','las-weather-fill',click)
    return () => { m.off('click','las-weather-fill',click) }
  }, [map,ready])
  useEffect(() => {
    const m=map.current
    if (!m || !ready) return
    const source=m.getSource('las-weather') as GeoJSONSource | undefined
    source?.setData({type:'FeatureCollection',features:show ? alerts.filter(f=>f.geometry).map(f=>({...f,geometry:f.geometry!,properties:{...f.properties,color:weatherColor(f.properties.event),alertID:f.id}})) : []})
  }, [map,ready,feed,now,show])
  const focus = (f: WeatherFeature) => {
    setSelected(f.id)
    const points: number[][]=[]
    function walk(g: GeoJSON.Geometry | null) {
      if (!g) return
      if (g.type==='GeometryCollection') { g.geometries.forEach(walk); return }
      const scan=(c: unknown) => { if (!Array.isArray(c)) return; if(typeof c[0]==='number') points.push(c as number[]); else c.forEach(scan) }
      scan(g.coordinates)
    }
    walk(f.geometry)
    if(points.length) map.current?.fitBounds([[Math.min(...points.map(p=>p[0])),Math.min(...points.map(p=>p[1]))],[Math.max(...points.map(p=>p[0])),Math.max(...points.map(p=>p[1]))]],{padding:50,maxZoom:11})
  }
  return <div className="absolute bottom-10 left-2 z-10 max-w-[min(25rem,85%)] text-xs" aria-label="NWS weather alerts">
    <button className={`btn h-9 shadow-lg ${stale ? 'border-amber-500 text-amber-200' : alerts.length ? 'border-rose-500 text-rose-200' : 'text-sky-200'}`} aria-expanded={open} onClick={()=>setOpen(v=>!v)}><CloudSnow size={15} aria-hidden /> NWS {stale ? 'unavailable / stale' : !feed ? 'loading…' : `${alerts.length} alert${alerts.length===1?'':'s'}`}</button>
    {open && <section className="panel mt-1 max-h-[45vh] space-y-2 overflow-y-auto rounded-lg bg-slate-950/95 p-3 shadow-xl">
      <div className="flex items-center justify-between gap-2"><b>NWS · Cass / Clay</b><button className="btn h-7" onClick={()=>setOpen(false)}>Close</button></div>
      <p className="text-slate-400">Cass County ND / Clay County MN · refreshes every minute</p>
      <label className="flex gap-2"><input type="checkbox" checked={show} onChange={e=>setShow(e.target.checked)} /> Show affected areas · warning red, watch gold, advisory blue</label>
      <p role="status" className={stale?'text-amber-200':'text-slate-400'}>{stale ? 'Weather coverage is not current. Check NWS directly.' : !feed ? 'Connecting to NWS…' : alerts.length ? 'Select an alert to view its area and instructions.' : 'No active NWS alerts for this coverage area.'} {feed?.checked && `Last checked ${new Date(feed.checked).toLocaleTimeString()}.`}</p>
      {alerts.map(f=><article key={f.id} className="rounded border border-slate-700 p-2">
        <button className="w-full text-left font-bold text-amber-100" onClick={()=>focus(f)}>{f.properties.event} · {f.properties.severity}</button>
        <p className="text-slate-300">{f.properties.areaDesc}</p>
        <p className="text-slate-400">{f.properties.onset && `Starts ${new Date(f.properties.onset).toLocaleString()} · `}Until {new Date(f.properties.ends || f.properties.expires).toLocaleString()}</p>
        {!f.geometry && <p className="text-amber-200">Boundary unavailable; alert remains in force for the named area.</p>}
        {selected===f.id && <div className="mt-2 space-y-2 whitespace-pre-wrap"><b>{f.properties.headline}</b><p>{f.properties.description}</p><p className="text-amber-100">{f.properties.instruction}</p><p>{f.properties.certainty} · {f.properties.urgency}</p></div>}
      </article>)}
      <a href="https://www.weather.gov/fgf/" target="_blank" rel="noreferrer" className="text-sky-300 underline">National Weather Service · Grand Forks</a>
    </section>}
  </div>
}
