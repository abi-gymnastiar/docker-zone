export function api(path, options) {
  return fetch(`/api${path}`, options).then(async (response) => {
    if (!response.ok) throw new Error(await response.text() || response.statusText)
    return response.status === 204 ? null : response.json()
  })
}

export function getLogs(serviceName) {
  return fetch(`/api/services/${serviceName}/logs`).then(async (response) => {
    if (!response.ok) throw new Error(await response.text() || response.statusText)
    return response.text()
  })
}

export function getPage(path, { page, pageSize, search = '' }) {
  const params = new URLSearchParams({ page: String(page), pageSize: String(pageSize) })
  if (search) params.set('search', search)
  return api(`${path}?${params}`)
}
