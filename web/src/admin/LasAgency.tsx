import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { api, errText } from '../api'
import { load } from '../store'
import { BRAND } from '../brand'
import { Page } from './Admin'

type Preview = { seq: number; unused: boolean; callTypes: { code: string; name: string; priority: number; runCard: { capability: string; count: number }[] | null }[] }

export default function LasAgency() {
  const [preview, setPreview] = useState<Preview | null>(null)
  const [confirmed, setConfirmed] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const refresh = () => api.get<Preview>('/admin/las').then(setPreview).catch(e => setError(errText(e)))
  useEffect(() => { refresh() }, [])
  const apply = async () => {
    if (!preview || !confirmed) return
    setBusy(true)
    try {
      await api.post('/admin/las', { seq: preview.seq, confirm: true })
      await load()
      await refresh()
      setConfirmed(false)
      toast.success('Lakes Area Security configuration applied')
    } catch (e) { setError(errText(e)); await refresh() }
    finally { setBusy(false) }
  }
  return <Page title="Lakes Area Security" intro={`${BRAND.legalName} · North Dakota private-security operations`}>
    <section className="panel mb-4 space-y-3 rounded-lg p-4 text-sm">
      <h2 className="font-semibold text-slate-50">One company. One operational workspace.</h2>
      <p>CAD coordinates calls, incidents and security units. MDT supports assigned officers. Records keep operational reports and their review history.</p>
      <p>Personnel, credentials, clients, contracts, billing and employee services remain in Management and NORTHSTAR. Accounts are currently local to this application; central account synchronization and the Neon records projection are not connected.</p>
      <a className="btn inline-flex" href={BRAND.management} target="_blank" rel="noopener noreferrer">Open Management</a>
      <p className="text-xs text-slate-400">LAS receives calls on its own lines. This system does not provide public emergency answering, government database access or public-safety dispatch authority. For emergencies, contact public emergency services.</p>
      <p className="text-xs text-slate-400">Powered by BitDispatcher · BioDream LLC · Apache 2.0</p>
    </section>
    <section className="panel space-y-3 rounded-lg p-4 text-sm">
      <h2 className="font-semibold text-slate-50">Apply the private-security starting configuration</h2>
      <p>Replaces the untouched upstream call-type catalog, unit types, mark-outs, qualifications and dispositions with LAS defaults. Sets Central time and disables the government message switch. Map settings, numbering, MFA policy, users, GIS, cameras and history are preserved. No station, officer or available unit is invented.</p>
      <p>Available before operational records or response plans exist. Existing units are preserved when their types are compatible with LAS. Custom call types require individual review. Review run cards and staffing before operational use. Take a volume backup first. Each change is recorded through the application ledger; this is not an atomic multi-event transaction, and an interrupted application can be retried.</p>
      {error && <p role="alert" className="text-rose-300">{error}</p>}
      {!preview ? <p>Loading configuration preview…</p> : <>
        <div className="overflow-x-auto"><table className="w-full text-left text-xs"><thead><tr><th>Code</th><th>Call type</th><th>Priority</th><th>Security units</th></tr></thead><tbody>{preview.callTypes.map(c => <tr key={c.code} className="border-t border-slate-800"><td className="py-2 font-mono">{c.code}</td><td>{c.name}</td><td>P{c.priority}</td><td>{c.runCard?.reduce((n, r) => n + r.count, 0) || 'Chosen by dispatcher'}</td></tr>)}</tbody></table></div>
        {!preview.unused && <p className="text-amber-300">Operational records or response plans already exist. Use individual configuration changes; bulk application is blocked.</p>}
        <label className="flex items-start gap-2"><input type="checkbox" checked={confirmed} disabled={!preview.unused || busy} onChange={e => setConfirmed(e.target.checked)} />I have reviewed this configuration and backed up the data volume.</label>
        <button className="btn btn-go" disabled={!preview.unused || !confirmed || busy} onClick={apply}>{busy ? 'Applying…' : 'Apply LAS configuration'}</button>
      </>}
    </section>
  </Page>
}
