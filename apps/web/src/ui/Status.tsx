import styles from './Status.module.css'

export type StatusTone = 'neutral' | 'info' | 'success' | 'warning' | 'danger'

export function Status({ tone, children }: { tone: StatusTone; children: string }) {
  return (
    <span className={`${styles.status} ${styles[tone]}`}>
      <span className={styles.dot} aria-hidden="true" />
      <span className={styles.label}>{children}</span>
    </span>
  )
}
