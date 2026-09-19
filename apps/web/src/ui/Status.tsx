import styles from './Status.module.css'
import type { StatusTone } from './tone'

export function Status({ tone, children }: { tone: StatusTone; children: string }) {
  return (
    <span className={`${styles.status} ${styles[tone]}`}>
      <span className={styles.dot} aria-hidden="true" />
      <span className={styles.label}>{children}</span>
    </span>
  )
}
