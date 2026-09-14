import { respond } from './client'

export type ProductStatus = 'active' | 'draft' | 'discontinued'

export type Product = {
  id: string
  sku: string
  name: string
  category: string
  priceCents: number
  stock: number
  status: ProductStatus
  updatedAt: string
  description: string
}

const PRODUCTS: Product[] = [
  {
    id: 'p-1041',
    sku: 'KTL-STN-500',
    name: 'Stoneware kettle, 0.5 L',
    category: 'Kitchen',
    priceCents: 6400,
    stock: 18,
    status: 'active',
    updatedAt: '2026-09-09T09:12:00Z',
    description: 'Hand-thrown stoneware kettle with a matte glaze. Dishwasher safe.',
  },
  {
    id: 'p-1042',
    sku: 'MUG-CLY-300',
    name: 'Clay mug, 300 ml',
    category: 'Kitchen',
    priceCents: 2200,
    stock: 3,
    status: 'active',
    updatedAt: '2026-09-11T14:40:00Z',
    description: 'Everyday mug in unglazed clay with a glazed interior.',
  },
  {
    id: 'p-1043',
    sku: 'TWL-LIN-NAT',
    name: 'Linen tea towel, natural',
    category: 'Textiles',
    priceCents: 1400,
    stock: 0,
    status: 'active',
    updatedAt: '2026-09-12T08:05:00Z',
    description: 'Stonewashed European linen, 50 x 70 cm.',
  },
  {
    id: 'p-1044',
    sku: 'BRD-OAK-LRG',
    name: 'Oak serving board, large',
    category: 'Kitchen',
    priceCents: 8900,
    stock: 11,
    status: 'draft',
    updatedAt: '2026-09-05T16:20:00Z',
    description: 'Single-piece oak board finished with food-safe oil.',
  },
  {
    id: 'p-1045',
    sku: 'CDL-BEE-SET',
    name: 'Beeswax candles, set of six',
    category: 'Home',
    priceCents: 3200,
    stock: 42,
    status: 'active',
    updatedAt: '2026-09-02T11:00:00Z',
    description: 'Pure beeswax dinner candles, eight-hour burn time.',
  },
  {
    id: 'p-1046',
    sku: 'APR-CAN-BLK',
    name: 'Canvas apron, black',
    category: 'Textiles',
    priceCents: 5600,
    stock: 7,
    status: 'discontinued',
    updatedAt: '2026-08-21T10:30:00Z',
    description: 'Heavy cotton canvas with adjustable neck strap.',
  },
]

export function listProducts(): Promise<Product[]> {
  return respond(PRODUCTS, [])
}

export function getProduct(id: string): Promise<Product | undefined> {
  return respond(PRODUCTS.find((product) => product.id === id))
}

export const productStatusLabels: Record<ProductStatus, string> = {
  active: 'Active',
  draft: 'Draft',
  discontinued: 'Discontinued',
}
