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
