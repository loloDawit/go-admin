import { useEffect, useRef, useState } from 'react'
import { isApiError } from './client'
import type { ApiError } from './client'

export type Resource<T> = {
  status: 'loading' | 'ready' | 'error'
  data: T | undefined
  error: ApiError | undefined
  reload: () => void
}

type State<T> = Omit<Resource<T>, 'reload'>

const LOADING: State<never> = { status: 'loading', data: undefined, error: undefined }

function toApiError(cause: unknown): ApiError {
  return isApiError(cause) ? cause : { code: 'unknown', message: 'Something went wrong.' }
}

export function useResource<T>(key: string, load: () => Promise<T>): Resource<T> {
  const [attempt, setAttempt] = useState(0)
  const [settled, setSettled] = useState<{ key: string; attempt: number; state: State<T> }>()
  const latest = useRef(load)

  useEffect(() => {
    latest.current = load
  })

  useEffect(() => {
    let active = true
    latest
      .current()
      .then((data) => {
        if (active) setSettled({ key, attempt, state: { status: 'ready', data, error: undefined } })
      })
      .catch((cause: unknown) => {
        if (!active) return
        setSettled({
          key,
          attempt,
          state: { status: 'error', data: undefined, error: toApiError(cause) },
        })
      })
    return () => {
      active = false
    }
  }, [key, attempt])

  const state = settled?.key === key && settled.attempt === attempt ? settled.state : LOADING

  return { ...state, reload: () => setAttempt((value) => value + 1) }
}
