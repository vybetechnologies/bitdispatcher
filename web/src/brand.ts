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

// Native BitDispatcher duty templates; Management grants remain independent.
export const SECURITY_ROLE_PRESETS: [string, string[]][] = [
  ['Owner / command administrator', ['admin', 'supervisor', 'dispatcher', 'calltaker', 'police', 'records', 'field']],
  ['Dispatch supervisor', ['supervisor', 'dispatcher', 'calltaker', 'police']],
  ['Dispatcher', ['dispatcher', 'calltaker', 'police']],
  ['Operations supervisor + MDT', ['supervisor', 'dispatcher', 'records', 'field']],
  ['Security officer / MDT', ['field']],
  ['Records staff', ['records']],
  ['Read-only observer', ['viewer']],
]
