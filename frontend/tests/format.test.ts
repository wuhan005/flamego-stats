import assert from 'node:assert/strict'
import { test } from 'node:test'

import { compactNumber, percent, relativeTime } from '../src/utils/format.ts'
import { languageColor } from '../src/utils/languages.ts'
import { moduleShortName, moduleURL } from '../src/utils/modules.ts'

test('relativeTime picks the largest unit', () => {
  const now = Date.parse('2026-10-07T12:00:00Z')
  assert.equal(relativeTime('2026-10-07T11:59:30Z', 'en', now), 'now')
  assert.equal(relativeTime('2026-10-07T09:00:00Z', 'en', now), '3 hours ago')
  assert.equal(relativeTime('2026-10-06T12:00:00Z', 'en', now), 'yesterday')
  assert.equal(relativeTime('2025-10-01T12:00:00Z', 'en', now), 'last year')
})

test('compactNumber and percent', () => {
  assert.equal(compactNumber(47860, 'en'), '47.9K')
  assert.equal(compactNumber(12, 'en'), '12')
  assert.equal(percent(1, 3), 33.3)
  assert.equal(percent(1, 0), 0)
})

test('moduleURL links the documented middleware to flamego.dev', () => {
  assert.equal(moduleShortName('github.com/flamego/session'), 'session')
  assert.equal(moduleShortName('github.com/flamego/template/v2'), 'template')
  assert.equal(moduleURL('github.com/flamego/session'), 'https://flamego.dev/middleware/session/')
  assert.equal(moduleURL('github.com/flamego/validator'), 'https://github.com/flamego/validator')
  assert.equal(moduleURL('github.com/flamego/template/v2'), 'https://flamego.dev/middleware/template/')
})

test('languageColor is stable for the unknown languages', () => {
  assert.equal(languageColor('Go'), '#00ADD8')
  assert.equal(languageColor('Brainfuck'), languageColor('Brainfuck'))
})
