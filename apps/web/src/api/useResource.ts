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

export function useResource<T>(load: () => Promise<T>, deps: unknown[]): Resource<T> {
  const [attempt, setAttempt] = useState(0)
  const [state, setState] = useState<State<T>>(LOADING)
  const latest = useRef(load)
  latest.current = load

  useEffect(() => {
    let active = true
    setState(LOADING)
    latest
      .current()
      .then((data) => {
        if (active) setState({ status: 'ready', data, error: undefined })
      })
      .catch((cause: unknown) => {
        if (!active) return
        const error: ApiError = isApiError(cause)
          ? cause
          : { code: 'unknown', message: 'Something went wrong.' }
        setState({ status: 'error', data: undefined, error })
      })
    return () => {
      active = false
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [attempt, ...deps])

  return { ...state, reload: () => setAttempt((value) => value + 1) }
}
