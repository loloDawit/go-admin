import type { ReactNode } from 'react'
import styles from './Alert.module.css'

export type AlertTone = 'info' | 'success' | 'warning' | 'danger'

export function Alert({
  tone = 'info',
  title,
  children,
  actions,
}: {
  tone?: AlertTone
  title: string
  children?: ReactNode
  actions?: ReactNode
}) {
  return (
    <div
      className={`${styles.alert} ${styles[tone]}`}
      role={tone === 'danger' ? 'alert' : 'status'}
    >
      <p className={styles.title}>{title}</p>
      {children && <div className={styles.body}>{children}</div>}
      {actions && <div className={styles.actions}>{actions}</div>}
    </div>
  )
}
