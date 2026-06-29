// Test suite para cloudflare/worker.js
// Ejecutar: node test-worker.js
//
// Valida la lógica HMAC y el armado del .lic sin necesidad de
// infraestructura Cloudflare ni Supabase.

const TEST_SECRET = 'test-secret-123';

// ─── Replicar funciones del worker ─────────────────────

async function computeHmac(payload, secret) {
  const encoder = new TextEncoder();
  const key = await crypto.subtle.importKey(
    'raw',
    encoder.encode(secret),
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign']
  );
  const signature = await crypto.subtle.sign('HMAC', key, encoder.encode(payload));
  return bytesToHex(new Uint8Array(signature));
}

function bytesToHex(bytes) {
  return Array.from(bytes)
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('');
}

async function buildLicFile(codigo, hw_id, issued, secret) {
  const payload = codigo + hw_id + issued;
  const hmac = await computeHmac(payload, secret);
  return { codigo, hw_id, issued, hmac };
}

// ─── Tests ──────────────────────────────────────────────

let passed = 0;
let failed = 0;

async function test(name, fn) {
  try {
    await fn();
    passed++;
    console.log(`  ✅ ${name}`);
  } catch (err) {
    failed++;
    console.log(`  ❌ ${name}: ${err.message}`);
  }
}

function assert(condition, msg) {
  if (!condition) throw new Error(msg || 'assertion failed');
}

function assertEqual(a, b, msg) {
  if (a !== b) throw new Error(msg || `expected ${JSON.stringify(a)} to equal ${JSON.stringify(b)}`);
}

// ─── Suite ──────────────────────────────────────────────

console.log('\n--- Worker Tests ---\n');

(async () => {
  // HMAC
  await test('HMAC produce un hash de 64 caracteres hex', async () => {
    const hmac = await computeHmac('test', TEST_SECRET);
    assertEqual(hmac.length, 64, `expected 64 hex chars, got ${hmac.length}: ${hmac}`);
  });

  await test('HMAC es determinista (mismo input = mismo output)', async () => {
    const a = await computeHmac('abc123', TEST_SECRET);
    const b = await computeHmac('abc123', TEST_SECRET);
    assertEqual(a, b);
  });

  await test('HMAC cambia si cambia el payload', async () => {
    const a = await computeHmac('payload-a', TEST_SECRET);
    const b = await computeHmac('payload-b', TEST_SECRET);
    assert(a !== b, 'HMAC debería diferir con distinto payload');
  });

  await test('HMAC cambia si cambia el secreto', async () => {
    const a = await computeHmac('test', TEST_SECRET);
    const b = await computeHmac('test', 'otro-secret');
    assert(a !== b, 'HMAC debería diferir con distinto secreto');
  });

  // Build .lic
  await test('buildLicFile produce el formato esperado', async () => {
    const lic = await buildLicFile('COD-123', 'hw-sha256-abc', '2026-06-28T12:00:00Z', TEST_SECRET);
    assert(lic.codigo, 'falta codigo');
    assert(lic.hw_id, 'falta hw_id');
    assert(lic.issued, 'falta issued');
    assert(lic.hmac, 'falta hmac');
    assertEqual(lic.codigo, 'COD-123');
    assertEqual(lic.hw_id, 'hw-sha256-abc');
  });

  await test('buildLicFile produce HMAC válido en el .lic', async () => {
    const lic = await buildLicFile('TPD-ABCD-1234', 'hw-fingerprint-1', '2026-07-01T00:00:00Z', TEST_SECRET);
    const expectedHmac = await computeHmac(lic.codigo + lic.hw_id + lic.issued, TEST_SECRET);
    assertEqual(lic.hmac, expectedHmac, 'HMAC del .lic no coincide con el esperado');
  });

  await test('HMAC con secreto incorrecto no valida', async () => {
    const lic = await buildLicFile('COD-999', 'hw-xyz', '2026-06-28T00:00:00Z', 'real-secret');
    const wrongHmac = await computeHmac(lic.codigo + lic.hw_id + lic.issued, 'wrong-secret');
    assert(lic.hmac !== wrongHmac, 'deberían diferir');
  });

  // ─── Tests de campos del .lic ──────────────────────────

  await test('buildLicFile incluye todos los campos requeridos', async () => {
    const lic = await buildLicFile('TPD-TEST-0001-AAAA', 'hw-id-test-123', '2026-01-01T00:00:00Z', TEST_SECRET);
    assert('codigo' in lic, 'falta campo codigo');
    assert('hw_id' in lic, 'falta campo hw_id');
    assert('issued' in lic, 'falta campo issued');
    assert('hmac' in lic, 'falta campo hmac');
    assert(Object.keys(lic).length === 4, `campos inesperados: ${Object.keys(lic).join(', ')}`);
  });

  await test('HMAC del .lic es reproducible para revalidar', async () => {
    const codigo = 'TPD-REPR-0001-BBBB';
    const hw_id  = 'hw-fingerprint-reproducible';
    const issued = '2026-06-28T12:00:00.000Z';
    const lic1   = await buildLicFile(codigo, hw_id, issued, TEST_SECRET);
    const lic2   = await buildLicFile(codigo, hw_id, issued, TEST_SECRET);
    assertEqual(lic1.hmac, lic2.hmac, 'el HMAC debe ser reproducible con los mismos datos');
  });

  await test('HMAC cambia si el codigo cambia (anti-tamper)', async () => {
    const hw_id  = 'hw-antitamper-test';
    const issued = '2026-06-28T00:00:00.000Z';
    const licOriginal  = await buildLicFile('TPD-ORIG-0001-CCCC', hw_id, issued, TEST_SECRET);
    const licAlterado  = await buildLicFile('TPD-FAKE-9999-ZZZZ', hw_id, issued, TEST_SECRET);
    assert(licOriginal.hmac !== licAlterado.hmac, 'HMAC debe diferir si el codigo cambia');
  });

  console.log(`\nResultados: ${passed} passed, ${failed} failed\n`);
  process.exit(failed > 0 ? 1 : 0);
})();
