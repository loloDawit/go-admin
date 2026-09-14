import type { ReactNode } from 'react'
import styles from './Page.module.css'

export function PageHeader({
  title,
  description,
  breadcrumb,
  actions,
}: {
  title: string
  description?: string
  breadcrumb?: ReactNode
  actions?: ReactNode
}) {
  return (
    <header className={styles.header}>
      <div className={styles.headingGroup}>
        {breadcrumb && <div className={styles.breadcrumb}>{breadcrumb}</div>}
        <h1 className={styles.title}>{title}</h1>
        {description && <p className={styles.description}>{description}</p>}
      </div>
      {actions && <div className={styles.actions}>{actions}</div>}
    </header>
  )
}

export function PageStack({ children }: { children: ReactNode }) {
  return <div className={styles.stack}>{children}</div>
}

export function Section({
  title,
  description,
  actions,
  children,
}: {
  title?: string
  description?: string
  actions?: ReactNode
  children: ReactNode
}) {
  return (
    <section className={styles.section}>
      {title && (
        <div className={styles.header}>
          <div className={styles.headingGroup}>
            <h2 className={styles.sectionTitle}>{title}</h2>
            {description && <p className={styles.sectionDescription}>{description}</p>}
          </div>
          {actions && <div className={styles.actions}>{actions}</div>}
        </div>
      )}
      {children}
    </section>
  )
}

export function DefinitionList({ items }: { items: { term: string; value: ReactNode }[] }) {
  return (
    <dl className={styles.definitions}>
      {items.map((item) => (
        <div key={item.term}>
          <dt className={styles.definitionTerm}>{item.term}</dt>
          <dd className={styles.definitionValue}>{item.value}</dd>
        </div>
      ))}
    </dl>
  )
}
