import { useState } from 'react'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { Button } from '../ui'
import { useAuth } from '../api/auth'
import type { Permission } from '../api/identity'
import styles from './AppShell.module.css'

const NAV_GROUPS: { label: string; items: { to: string; label: string; end?: boolean; require?: Permission }[] }[] = [
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
      { to: '/staff', label: 'Staff', require: 'view_staff' },
      { to: '/roles', label: 'Roles', require: 'view_roles' },
      { to: '/permissions', label: 'Permissions', require: 'view_roles' },
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
  const auth = useAuth()
  const navigate = useNavigate()

  return (
    <div className={styles.shell}>
      <header className={styles.topbar}>
        <div className={styles.brand}>
          <span className={styles.brandName}>Northgate Supply</span>
          <span className={styles.brandContext}>Back office</span>
        </div>
        <div className={styles.topbarRight}>
          <span className={styles.account}>{auth.user?.email}</span>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => {
              void auth.signOut().then(() => navigate('/login', { replace: true }))
            }}
          >
            Sign out
          </Button>
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
          {NAV_GROUPS.map((group) => {
            const items = group.items.filter((item) => !item.require || auth.hasPermission(item.require))
            if (items.length === 0) return null
            return (
              <div className={styles.group} key={group.label}>
                <span className={styles.groupLabel}>{group.label}</span>
                {items.map((item) => (
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
            )
          })}
        </nav>

        <main className={styles.main}>
          <Outlet />
        </main>
      </div>
    </div>
  )
}
