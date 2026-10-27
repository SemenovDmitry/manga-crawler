// Событие «сессия не восстановлена»: refresh не удался — нужно на логин.
type UnauthorizedListener = () => void

const listeners = new Set<UnauthorizedListener>()

// Один неудачный refresh обычно сопровождается несколькими параллельными
// 401 (все они дожидались общего refresh в очереди). Каждому отвечать своим
// тостом и своей навигацией нельзя — получается шторм «сессия истекла»,
// поэтому событие гасим на коротком окне после предыдущего.
const EMIT_COOLDOWN_MS = 1_000
let lastEmitAt = 0

export function onUnauthorized(cb: UnauthorizedListener): () => void {
  listeners.add(cb)
  return () => {
    listeners.delete(cb)
  }
}

export function emitUnauthorized() {
  const now = Date.now()
  if (now - lastEmitAt < EMIT_COOLDOWN_MS) {
    return
  }
  lastEmitAt = now
  for (const cb of listeners) {
    cb()
  }
}
