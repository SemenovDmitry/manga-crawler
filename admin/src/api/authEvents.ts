// Событие «сессия не восстановлена»: refresh не удался — нужно на логин.
type UnauthorizedListener = () => void

const listeners = new Set<UnauthorizedListener>()

export function onUnauthorized(cb: UnauthorizedListener): () => void {
  listeners.add(cb)
  return () => {
    listeners.delete(cb)
  }
}

export function emitUnauthorized() {
  for (const cb of listeners) {
    cb()
  }
}
