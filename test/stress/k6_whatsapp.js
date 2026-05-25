/**
 * k6_whatsapp.js — Test de carga para el REST API de hermes-go.
 *
 * Simula N usuarios virtuales enviando mensajes concurrentes al agente.
 *
 * Uso:
 *   k6 run --vus 100 --duration 30s k6_whatsapp.js
 *
 * Variables de entorno:
 *   BASE_URL   URL base del servidor (default: http://localhost:8080)
 *   TOKEN      Bearer token (default: test-token)
 */

import http from 'k6/http'
import { check, sleep } from 'k6'
import { Counter, Trend } from 'k6/metrics'

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080'
const TOKEN = __ENV.TOKEN || 'test-token'

const errorCount = new Counter('errors')
const latency = new Trend('message_latency_ms', true)

export const options = {
  vus: 100,
  duration: '30s',
  thresholds: {
    http_req_failed: ['rate<0.01'],   // menos del 1% de errores
    http_req_duration: ['p(95)<2000'], // p95 < 2s
  },
}

export default function () {
  const sessionID = `stress_vus_${__VU}`
  const payload = JSON.stringify({
    session_id: sessionID,
    message: `Mensaje de prueba VU=${__VU} iter=${__ITER}`,
    user_id: `user_${__VU}`,
  })

  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${TOKEN}`,
  }

  const start = Date.now()
  const res = http.post(`${BASE_URL}/v1/chat`, payload, { headers })
  latency.add(Date.now() - start)

  const ok = check(res, {
    'status 2xx': (r) => r.status >= 200 && r.status < 300,
  })

  if (!ok) {
    errorCount.add(1)
  }

  sleep(0.1)
}
