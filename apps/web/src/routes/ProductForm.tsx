import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import {
  Alert,
  Button,
  CheckboxField,
  PageHeader,
  PageStack,
  Section,
  SelectField,
  TextField,
  TextareaField,
} from '../ui'
import { productStatusLabels } from '../api/catalog'
import type { ProductStatus } from '../api/catalog'
import styles from './ProductForm.module.css'

const STATUSES: ProductStatus[] = ['active', 'draft', 'discontinued']

export function ProductForm() {
  const navigate = useNavigate()
  const { productId } = useParams()
  const editing = Boolean(productId)
  const [name, setName] = useState(editing ? 'Clay mug, 300 ml' : '')
  const [submitting, setSubmitting] = useState(false)
  const [saved, setSaved] = useState(false)
  const [touched, setTouched] = useState(false)

  const nameError = touched && name.trim() === '' ? 'Give the product a name.' : undefined

  return (
    <PageStack>
      <PageHeader
        breadcrumb={<Link to="/products">Products</Link>}
        title={editing ? 'Edit product' : 'New product'}
        description="Everything here is visible to customers on the storefront."
      />

      {saved && (
        <Alert tone="success" title="Product saved">
          The change is local until the catalog service is connected.
        </Alert>
      )}

      <form
        className={styles.form}
        onSubmit={(event) => {
          event.preventDefault()
          setTouched(true)
          if (name.trim() === '') return
          setSubmitting(true)
          setTimeout(() => {
            setSubmitting(false)
            setSaved(true)
          }, 600)
        }}
      >
        <Section title="Details">
          <div className={styles.grid}>
            <TextField
              label="Name"
              value={name}
              error={nameError}
              onChange={(event) => setName(event.target.value)}
              onBlur={() => setTouched(true)}
              placeholder="Stoneware kettle, 0.5 L"
            />
            <TextField label="SKU" defaultValue={editing ? 'MUG-CLY-300' : ''} help="Uppercase, no spaces." />
            <SelectField label="Category" defaultValue="Kitchen">
              <option>Kitchen</option>
              <option>Textiles</option>
              <option>Home</option>
            </SelectField>
            <SelectField label="Status" defaultValue="draft">
              {STATUSES.map((status) => (
                <option key={status} value={status}>
                  {productStatusLabels[status]}
                </option>
              ))}
            </SelectField>
          </div>
          <TextareaField
            label="Description"
            optional
            placeholder="What the customer sees on the product page."
          />
        </Section>

        <Section title="Price and stock">
          <div className={styles.grid}>
            <TextField label="Price" inputMode="decimal" defaultValue={editing ? '22.00' : ''} help="In pounds." />
            <TextField label="Stock on hand" inputMode="numeric" defaultValue={editing ? '3' : '0'} />
            <TextField label="Reorder level" inputMode="numeric" defaultValue="3" optional />
          </div>
          <CheckboxField label="Keep selling when stock reaches zero" />
        </Section>

        <div className={styles.actions}>
          <Button type="submit" variant="primary" loading={submitting}>
            {editing ? 'Save changes' : 'Create product'}
          </Button>
          <Button variant="ghost" onClick={() => navigate('/products')}>
            Cancel
          </Button>
        </div>
      </form>
    </PageStack>
  )
}
