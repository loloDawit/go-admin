// The M5.5 product path under load: log in, create and activate a product,
// create a customer and an order, then advance it to paid.
//
// The threshold is a stated target, not a guess dressed up as a fact. If it
// fails, the measurement is the result: record the number and which span
// dominated the slowest traces.
//
// Every VU authenticates as the same member of staff, and the gateway's rate
// limiter is per principal, so the load run raises RATE_LIMIT_PER_SECOND. It is
// measuring the order path, not the limiter, which has its own test.
import http from 'k6/http'
import { check, fail } from 'k6'

const BASE = __ENV.BASE_URL || 'http://localhost:8080'

export const options = {
  vus: Number(__ENV.VUS || 20),
  // Without this k6 empties the cookie jar between iterations, which would
  // force a login per iteration and measure the login limiter instead.
  noCookiesReset: true,
  duration: __ENV.DURATION || '60s',
  thresholds: {
    'http_req_duration{name:create_order}': ['p(95)<300'],
    'checks{type:required}': ['rate==1.0'],
  },
}

const JSON_HEADERS = { 'Content-Type': 'application/json' }

function post(path, body, name) {
  return http.post(`${BASE}${path}`, JSON.stringify(body), {
    headers: JSON_HEADERS,
    tags: { name },
  })
}

// A load test that measures the latency of a 401 measures nothing, so every
// response is checked and a failure stops the iteration rather than skewing
// the numbers of the step after it.
function expect(response, status, what) {
  const ok = check(response, { [what]: (r) => r.status === status }, { type: 'required' })
  if (!ok) fail(`${what}: got ${response.status}`)
  return response.json()
}

// One login per VU, as a member of staff actually works: noCookiesReset keeps
// the session for the rest of the run.
export default function () {
  if (__ITER === 0) {
    expect(post('/api/v1/login', {
      email: __ENV.OWNER_EMAIL || 'owner@example.com',
      password: __ENV.OWNER_PASSWORD || 'dev_only_owner_password',
    }, 'login'), 200, 'logged in')
  }

  const unique = `${__VU}-${__ITER}-${Date.now()}`

  const product = expect(post('/api/v1/products', {
    sku: `K6-${unique}`,
    title: `Load ${unique}`,
    description: 'load test',
    priceMinor: 2500,
  }, 'create_product'), 201, 'product created')

  expect(post(`/api/v1/products/${product.id}/activate`, null, 'activate_product'), 200, 'product activated')

  const customer = expect(post('/api/v1/customers', {
    email: `k6-${unique}@example.com`,
    name: `Load ${unique}`,
  }, 'create_customer'), 201, 'customer created')

  const order = expect(post('/api/v1/orders', {
    customerId: customer.id,
    items: [{ productId: product.id, quantity: 1 }],
  }, 'create_order'), 201, 'order created')

  expect(post(`/api/v1/orders/${order.id}/status`, { status: 'paid' }, 'advance_order'), 200, 'order paid')
}
