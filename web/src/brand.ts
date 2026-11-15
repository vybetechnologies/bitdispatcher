// LAS is one North Dakota private-security organization. Compatibility keys stay internal.
export const BRAND = {
  governmentConnections: false, name: 'Lakes Area Security', legalName: 'Lakes Area Security Corporation', product: 'LAS Operations',
  scope: 'Private security · North Dakota', management: 'https://mgt.lakesareasecurity.com',
  notice: 'AUTHORIZED USE ONLY — Lakes Area Security Corporation. Access and activity are recorded.',
}
export const SECURITY_DISCIPLINES = [{ value: 'police', label: 'Security' }, { value: 'other', label: 'Administrative / support' }]
export const SECURITY_DUTIES: [string, string][] = [
  ['calltaker', 'Call intake'], ['police', 'Security dispatcher'], ['dispatcher', 'Dispatcher'],
  ['supervisor', 'Supervisor'], ['admin', 'Administrator'], ['records', 'Records'], ['field', 'Security officer (MDT)'], ['viewer', 'Viewer'],
]
export const disciplineLabel = (key: string) => SECURITY_DISCIPLINES.find(d => d.value === key)?.label ?? `${key} (historical)`
export const UNAVAILABLE_ADMIN = new Set(['agencies', 'facilities', 'publicalerts', 'pip'])
