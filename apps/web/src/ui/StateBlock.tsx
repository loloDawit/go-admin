import type { ReactNode } from 'react'
import styles from './StateBlock.module.css'

export function StateBlock({
  tone = 'neutral',
  title,
  description,
  action,
  centered = false,
}: {
  tone?: 'neutral' | 'error'
  title: string
  description?: string
  action?: ReactNode
  centered?: boolean
}) {
  return (
    <div
      className={`${styles.block} ${centered ? styles.centered : ''}`}
      role={tone === 'error' ? 'alert' : undefined}
    >
      <p className={`${styles.title} ${tone === 'error' ? styles.errorTitle : ''}`}>{title}</p>
      {description && <p className={styles.description}>{description}</p>}
      {action && <div className={styles.action}>{action}</div>}
    </div>
  )
}
