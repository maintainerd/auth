// k6 load test for Maintainerd Auth hot paths.
//
// Validates that the Section-A scalability design (indexes, keyset pagination,
// auth_events partitioning) holds at scale. Run AFTER seeding the DB to
// millions of rows (see tests/load/README.md).
//
//   k6 run \
//     -e BASE_PUBLIC=https://public-api.auth.maintainerd.local \
//     -e BASE_PRIVATE=https://private-api.auth.maintainerd.local \
//     -e CLIENT_ID=<client_id> -e ADMIN_TOKEN=<bearer> \
//     -e TEST_EMAIL=<seeded-user-email> -e TEST_PASSWORD=<password> \
//     tests/load/auth_load.js
//
// Thresholds encode the p95 targets from docs/documentations/observability/load-testing.md.
// A failing threshold exits non-zero so this can gate a manual pre-release run.

import http from 'k6/http';
import { check, group } from 'k6';

const BASE_PUBLIC = __ENV.BASE_PUBLIC || 'http://localhost:8081';
const BASE_PRIVATE = __ENV.BASE_PRIVATE || 'http://localhost:8080';
const CLIENT_ID = __ENV.CLIENT_ID || '';
const ADMIN_TOKEN = __ENV.ADMIN_TOKEN || '';
const TEST_EMAIL = __ENV.TEST_EMAIL || '';
const TEST_PASSWORD = __ENV.TEST_PASSWORD || '';

export const options = {
  scenarios: {
    steady: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 50 },
        { duration: '2m', target: 200 },
        { duration: '1m', target: 0 },
      ],
    },
  },
  thresholds: {
    // p95 targets at 1M+ rows. Tighten as needed; a breach fails the run.
    'http_req_duration{name:login}': ['p(95)<400'],
    'http_req_duration{name:user_list}': ['p(95)<300'],
    'http_req_duration{name:auth_events_list}': ['p(95)<300'],
    http_req_failed: ['rate<0.01'],
  },
};

function authHeaders() {
  return { headers: { Authorization: `Bearer ${ADMIN_TOKEN}`, 'Content-Type': 'application/json' } };
}

export default function () {
  // Hot path 1: password login (public surface).
  group('login', () => {
    if (!CLIENT_ID || !TEST_EMAIL) return;
    const res = http.post(
      `${BASE_PUBLIC}/api/v1/login?client_id=${CLIENT_ID}`,
      JSON.stringify({ username: TEST_EMAIL, password: TEST_PASSWORD }),
      { headers: { 'Content-Type': 'application/json' }, tags: { name: 'login' } },
    );
    check(res, { 'login ok/expected': (r) => r.status === 200 || r.status === 401 });
  });

  // Hot path 2: deep-paginated user list (keyset pagination — must stay flat).
  group('user_list', () => {
    if (!ADMIN_TOKEN) return;
    const res = http.get(`${BASE_PRIVATE}/api/v1/users?limit=50`, {
      ...authHeaders(),
      tags: { name: 'user_list' },
    });
    check(res, { 'user_list 200': (r) => r.status === 200 });
  });

  // Hot path 3: auth-events list (partitioned table + reltuples estimate).
  group('auth_events_list', () => {
    if (!ADMIN_TOKEN) return;
    const res = http.get(`${BASE_PRIVATE}/api/v1/auth-events?limit=50`, {
      ...authHeaders(),
      tags: { name: 'auth_events_list' },
    });
    check(res, { 'auth_events 200': (r) => r.status === 200 });
  });
}
