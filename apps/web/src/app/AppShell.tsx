import { useState } from 'react'
import { NavLink, Outlet } from 'react-router-dom'
import { Button } from '../ui'
import styles from './AppShell.module.css'

const NAV_GROUPS = [
  {
    label: 'Shop',
    items: [
      { to: '/', label: 'Dashboard', end: true },
      { to: '/orders', label: 'Orders' },
      { to: '/products', label: 'Products' },
      { to: '/customers', label: 'Customers' },
    ],
  },
  {
    label: 'Access',
    items: [
      { to: '/staff', label: 'Staff' },
      { to: '/roles', label: 'Roles' },
      { to: '/permissions', label: 'Permissions' },
    ],
  },
  {
    label: 'Account',
    items: [
      { to: '/profile', label: 'Profile' },
      { to: '/settings', label: 'Settings' },
    ],
  },
]

export function AppShell() {
  const [navOpen, setNavOpen] = useState(false)

  return (
    <div className={styles.shell}>
      <header className={styles.topbar}>
        <div className={styles.brand}>
          <span className={styles.brandName}>Northgate Supply</span>
          <span className={styles.brandContext}>Back office</span>
        </div>
        <div className={styles.topbarRight}>
          <span className={styles.account}>Mara Lindqvist · Owner</span>
          <Button
            variant="secondary"
            size="sm"
            className={styles.menuButton}
            aria-expanded={navOpen}
            aria-controls="app-nav"
            onClick={() => setNavOpen((open) => !open)}
          >
            {navOpen ? 'Close' : 'Menu'}
          </Button>
        </div>
      </header>

      <div className={styles.body}>
        <nav
          id="app-nav"
          aria-label="Sections"
          className={`${styles.sidebar} ${navOpen ? '' : styles.sidebarHidden}`}
        >
          {NAV_GROUPS.map((group) => (
            <div className={styles.group} key={group.label}>
              <span className={styles.groupLabel}>{group.label}</span>
              {group.items.map((item) => (
                <NavLink
                  key={item.to}
                  to={item.to}
                  end={item.end}
                  onClick={() => setNavOpen(false)}
                  className={({ isActive }) =>
                    `${styles.link} ${isActive ? styles.linkActive : ''}`
                  }
                >
                  {item.label}
                </NavLink>
              ))}
            </div>
          ))}
        </nav>

        <main className={styles.main}>
          <Outlet />
        </main>
      </div>
    </div>
  )
}
