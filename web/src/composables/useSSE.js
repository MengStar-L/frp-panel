import { onBeforeUnmount } from 'vue'

// useSSE manages a single EventSource that is closed automatically when the
// component unmounts. handlers maps event names to (data, event) callbacks;
// the key "error" maps to the EventSource onerror handler.
export function useSSE() {
  let es = null

  function close() {
    if (es) {
      es.close()
      es = null
    }
  }

  function open(url, handlers = {}) {
    close()
    es = new EventSource(url)
    for (const [event, fn] of Object.entries(handlers)) {
      if (event === 'error') {
        es.onerror = fn
      } else {
        es.addEventListener(event, (e) => fn(e.data, e))
      }
    }
    return es
  }

  onBeforeUnmount(close)
  return { open, close }
}
