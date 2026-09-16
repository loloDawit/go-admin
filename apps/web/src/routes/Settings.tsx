import { Link } from 'react-router-dom'
import { DefinitionList, PageHeader, PageStack, Section, StateBlock } from '../ui'
import { useAuth } from '../api/auth'

export function Settings() {
  const { user } = useAuth()

  return (
    <PageStack>
      <PageHeader title="Settings" description="What this back office is configured with." />

      <Section
        title="Your account"
        description="Change your password from your profile."
        actions={<Link to="/profile">Profile</Link>}
      >
        <DefinitionList
          items={[
            { term: 'Signed in as', value: user?.email ?? '—' },
            { term: 'Permissions', value: user?.permissions.join(', ') || 'None' },
          ]}
        />
      </Section>

      <Section title="Shop settings">
        <StateBlock
          title="Nothing to configure yet"
          description="Shop-wide settings need a service to store them; none exists. This screen will list them when one does."
        />
      </Section>
    </PageStack>
  )
}
