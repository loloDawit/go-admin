import { useLocation } from 'react-router-dom'
import { scenarioFromSearch } from '../api/client'
import type { Scenario } from '../api/client'

export function useScenario(): Scenario {
  return scenarioFromSearch(useLocation().search)
}
