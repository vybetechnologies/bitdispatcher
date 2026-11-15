import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { api, errText } from '../api'
import { Field } from '../ui'
import { Page } from './Admin'

type Config = { bitvoiceUrl: string; bitvoiceFingerprint: string; bitvoiceTokenSet: boolean }
export default function LasIntegrations() {
  const [config, setConfig] = useState<Config | null>(null)
  const [token, setToken] = useState('')
  useEffect(() => { api.get<Config>('/admin/integrations').then(setConfig).catch(e => toast.error(errText(e))) }, [])
  const save = async () => {
    if (!config) return
    try {
      const next = await api.put<Config>('/admin/integrations', { bitvoiceUrl: config.bitvoiceUrl, bitvoiceFingerprint: config.bitvoiceFingerprint, bitvoiceToken: token, eidoToken: '-', radioSecret: '-', aliListen: '' })
      setConfig(next); setToken(''); toast.success('Phone connection saved')
    } catch (e) { toast.error(errText(e)) }
  }
  return <Page title="Connections" intro="Private-security connections for Lakes Area Security.">
    <section className="panel space-y-3 rounded-lg p-4 text-sm">
      <h2 className="font-semibold">BitVoice — company phone system</h2>
      <p>Connect only an authorized company phone system. A new DID, trunk, hosting and provider setup are separate work. Saving this connection does not provision a number or port the company's existing number.</p>
      {!config ? <p>Loading…</p> : <>
        <Field label="BitVoice HTTPS address" htmlFor="las-bv-url"><input id="las-bv-url" className="input" value={config.bitvoiceUrl} onChange={e => setConfig({ ...config, bitvoiceUrl: e.target.value })} /></Field>
        <Field label="Certificate SHA-256 fingerprint (for a self-signed server)" htmlFor="las-bv-pin"><input id="las-bv-pin" className="input font-mono" value={config.bitvoiceFingerprint} onChange={e => setConfig({ ...config, bitvoiceFingerprint: e.target.value })} /></Field>
        <Field label="API credential" htmlFor="las-bv-token" hint={config.bitvoiceTokenSet ? 'Configured. Leave blank to keep; a dash clears it.' : 'Not configured.'}><input id="las-bv-token" type="password" autoComplete="off" className="input" value={token} onChange={e => setToken(e.target.value)} /></Field>
        <button className="btn btn-go" onClick={save}>Save phone connection</button>
      </>}
    </section>
    <section className="panel mt-4 space-y-2 rounded-lg p-4 text-sm"><h2 className="font-semibold">Separate connections</h2><p>Paging gateways are managed in Paging groups. Traffic feeds are managed in Road closures &amp; traffic. Cameras and county GIS have their own pages.</p><p>Management account synchronization, a Neon records projection, NDCOURTS and CLEAR are not connected. Government-only interfaces remain unavailable.</p></section>
  </Page>
}
