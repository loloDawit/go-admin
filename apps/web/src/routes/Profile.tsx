import { useNavigate } from 'react-router-dom'
import { Button, DefinitionList, PageHeader, PageStack, Section } from '../ui'
import { useAuth } from '../api/auth'
import { permissionLabels } from '../api/identity'

export function Profile() {
  const auth = useAuth()
  const navigate = useNavigate()
  const user = auth.user

  return (
    <PageStack>
      <PageHeader title="Profile" description="Your own sign-in details." />

      <Section title="Account">
        <DefinitionList
          items={[
            { term: 'Email', value: user?.email ?? '—' },
            {
              term: 'Permissions',
              value: user ? user.permissions.map((permission) => permissionLabels[permission]).join(', ') : '—',
            },
          ]}
        />
      </Section>

      <Section title="Password" description="Choose a new password for this account.">
        <Button variant="secondary" onClick={() => navigate('/change-password')}>
          Change password
        </Button>
      </Section>
    </PageStack>
  )
}
