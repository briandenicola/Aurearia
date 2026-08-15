import type { Page, Route } from '@playwright/test'
import { expect } from '@playwright/test'
import {
  buildRomanDenariusCore,
  buildTestCoinSets,
  buildTestStorageLocations,
  buildTestTags,
} from '../../src/test/fixtures'
import type { Coin, CoinImage, CoinListResponse, CoinMutationPayload, CoinSet, StorageLocation, Tag, UserInfo } from '../../src/types'

export const workflowUser = {
  id: 101,
  username: 'workflow-user',
  role: 'admin',
  email: 'workflow@example.test',
  avatarPath: '',
  isPublic: false,
  bio: '',
  zipCode: '',
  numisBidsUsername: '',
  numisBidsConfigured: false,
  pushoverEnabled: false,
  coinOfDayEnabled: true,
} as const

export interface WorkflowApiState {
  coins: Coin[]
  storageLocations: StorageLocation[]
  tags: Tag[]
  sets: CoinSet[]
  createPayloads: CoinMutationPayload[]
  updatePayloads: Array<{ id: number; payload: CoinMutationPayload }>
  tagPayloads: Array<{ action: 'add' | 'remove'; coinId: number; tagId: number }>
  setPayloads: Array<{ action: 'add' | 'remove'; coinId: number; setId: number }>
  imageUploads: Array<{ coinId: number; imageType: string; isPrimary: boolean; fileName: string; contentType: string }>
  imageDeletes: Array<{ coinId: number; imageId: number }>
  mediaRequests: Array<{ path: string; authorization: string; cacheControl: string }>
  coinQueries: Array<Record<string, string>>
  authorizedRequests: string[]
  deepIdentificationJobs: Array<{ id: number; notes: string; providers: string; status: string; source?: string; report?: unknown; proposal?: Record<string, { proposed: unknown; ownerEdited: boolean; ownerValue: unknown; accepted: boolean | null }>; appliedAt?: string | null; appliedCoinId?: number | null; appliedDraftId?: number | null }>
  deepIdentificationCancels: number[]
  deepIdentificationProposalUpdates: Array<{ id: number; fields: Record<string, unknown> }>
  deepIdentificationApplies: Array<{ id: number; target: string }>
}

export async function installAuthenticatedSession(page: Page) {
  await page.addInitScript((user) => {
    window.localStorage.setItem('token', 'workflow-access-token')
    window.localStorage.setItem('refreshToken', 'workflow-refresh-token')
    window.localStorage.setItem('user', JSON.stringify(user))
  }, workflowUser)
}

export async function installWorkflowApiMocks(page: Page, initialCoins: Coin[] = [buildRomanDenariusCore()]): Promise<WorkflowApiState> {
  const state: WorkflowApiState = {
    coins: initialCoins.map((coin) => cloneCoin(coin)),
    storageLocations: buildTestStorageLocations(),
    tags: buildTestTags(),
    sets: buildTestCoinSets(),
    createPayloads: [],
    updatePayloads: [],
    tagPayloads: [],
    setPayloads: [],
    imageUploads: [],
    imageDeletes: [],
    mediaRequests: [],
    coinQueries: [],
    authorizedRequests: [],
    deepIdentificationJobs: [],
    deepIdentificationCancels: [],
    deepIdentificationProposalUpdates: [],
    deepIdentificationApplies: [],
  }
  let nextCoinId = 7001
  let nextImageId = 9001
  let nextDeepJobId = 5001

  await page.route('**/uploads/**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    if (url.pathname.startsWith('/api/uploads/')) {
      await media(route, state)
      return
    }
    await route.fulfill({
      status: 204,
    })
  })

  await page.route('**/api/**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    if (!url.pathname.startsWith('/api/')) {
      await route.continue()
      return
    }
    const path = url.pathname.replace(/^\/api/, '')
    const method = request.method()
    const authorization = request.headers()['authorization']
    if (authorization) state.authorizedRequests.push(`${method} ${path}`)

    if (path === '/auth/login' && method === 'POST') {
      await json(route, {
        token: 'workflow-access-token',
        refreshToken: 'workflow-refresh-token',
        user: workflowUser,
      })
      return
    }

    if (path === '/auth/me' && method === 'GET') {
      await json(route, {
        ...workflowUser,
        createdAt: '2024-01-01T00:00:00Z',
        emailMissing: false,
      } satisfies UserInfo)
      return
    }

    if (path === '/notifications/unread-count' && method === 'GET') {
      await json(route, { count: 0 })
      return
    }

    if (path.startsWith('/uploads/') && method === 'GET') {
      await media(route, state)
      return
    }

    if (path === '/admin/settings' && method === 'GET') {
      await json(route, {
        CoinCategories: 'Roman\nGreek\nByzantine\nModern\nOther',
        CoinEras: 'ancient\nmedieval\nmodern',
      })
      return
    }

    if (path === '/storage-locations' && method === 'GET') {
      await json(route, { storageLocations: state.storageLocations })
      return
    }

    if (path === '/tags' && method === 'GET') {
      await json(route, { tags: state.tags })
      return
    }

    if (path === '/sets' && method === 'GET') {
      await json(route, {
        sets: state.sets.map((set) => ({
          id: set.id,
          name: set.name,
          color: set.color,
          icon: set.icon,
          setType: set.setType,
          coinCount: state.coins.filter((coin) => coin.sets?.some((coinSet) => coinSet.id === set.id)).length,
          totalValue: state.coins
            .filter((coin) => coin.sets?.some((coinSet) => coinSet.id === set.id))
            .reduce((sum, coin) => sum + (coin.currentValue ?? 0), 0),
        })),
      })
      return
    }

    if (path === '/suggestions' && method === 'GET') {
      await json(route, [])
      return
    }

    if (path === '/coins' && method === 'GET') {
      const filteredCoins = filterCoins(state.coins, url.searchParams)
      state.coinQueries.push(Object.fromEntries(url.searchParams.entries()))
      await json(route, {
        coins: filteredCoins,
        total: filteredCoins.length,
        page: Number(url.searchParams.get('page') ?? 1),
        limit: Number(url.searchParams.get('limit') ?? 50),
      } satisfies CoinListResponse)
      return
    }

    if (path === '/coins' && method === 'POST') {
      const payload = request.postDataJSON() as CoinMutationPayload
      state.createPayloads.push(payload)
      const created = buildRomanDenariusCore({
        ...payload,
        id: nextCoinId++,
        userId: workflowUser.id,
        images: [],
        tags: [],
        sets: [],
        references: [],
      })
      state.coins.push(created)
      await json(route, created)
      return
    }

    const imageUploadMatch = path.match(/^\/coins\/(\d+)\/images$/)
    if (imageUploadMatch && method === 'POST') {
      const coinId = Number(imageUploadMatch[1])
      const body = (await request.postDataBuffer())?.toString('utf8') ?? ''
      const imageType = body.match(/name="imageType"\r?\n\r?\n([^\r\n-]+)/)?.[1]?.trim() || 'other'
      const isPrimary = body.match(/name="isPrimary"\r?\n\r?\n([^\r\n-]+)/)?.[1]?.trim() === 'true'
      const fileName = body.match(/filename="([^"]+)"/)?.[1] ?? 'workflow-upload.png'
      const uploadedImage: CoinImage = {
        id: nextImageId++,
        coinId,
        filePath: `test-fixtures/${coinId}-${imageType}-${nextImageId}.png`,
        imageType: imageType as CoinImage['imageType'],
        isPrimary,
        createdAt: '2026-06-09T00:00:00Z',
      }
      state.imageUploads.push({
        coinId,
        imageType,
        isPrimary,
        fileName,
        contentType: request.headers()['content-type'] ?? '',
      })
      state.coins = state.coins.map((coin) => coin.id === coinId
        ? cloneCoin({ ...coin, images: [...coin.images, uploadedImage] })
        : coin)
      await json(route, uploadedImage)
      return
    }

    const imageDeleteMatch = path.match(/^\/coins\/(\d+)\/images\/(\d+)$/)
    if (imageDeleteMatch && method === 'DELETE') {
      const coinId = Number(imageDeleteMatch[1])
      const imageId = Number(imageDeleteMatch[2])
      state.imageDeletes.push({ coinId, imageId })
      state.coins = state.coins.map((coin) => coin.id === coinId
        ? cloneCoin({ ...coin, images: coin.images.filter((image) => image.id !== imageId) })
        : coin)
      await json(route, { message: 'Image deleted' })
      return
    }

    const coinMatch = path.match(/^\/coins\/(\d+)$/)
    if (coinMatch && method === 'GET') {
      const id = Number(coinMatch[1])
      const coin = state.coins.find((item) => item.id === id)
      if (coin) {
        await json(route, coin)
      } else {
        await json(route, { error: 'Not found' }, 404)
      }
      return
    }

    if (coinMatch && method === 'PUT') {
      const id = Number(coinMatch[1])
      const payload = request.postDataJSON() as CoinMutationPayload
      state.updatePayloads.push({ id, payload })
      const current = state.coins.find((item) => item.id === id) ?? buildRomanDenariusCore({ id })
      const storageLocationId = payload.storageLocationId === undefined ? current.storageLocationId : payload.storageLocationId
      const storageLocation = storageLocationId == null
        ? null
        : state.storageLocations.find((location) => location.id === storageLocationId) ?? null
      const updated = cloneCoin({
        ...current,
        ...payload,
        id,
        storageLocationId,
        storageLocation: storageLocation ? { id: storageLocation.id, name: storageLocation.name } : null,
      })
      state.coins = state.coins.map((item) => item.id === id ? updated : item)
      await json(route, updated)
      return
    }

    const addTagMatch = path.match(/^\/coins\/(\d+)\/tags$/)
    if (addTagMatch && method === 'POST') {
      const coinId = Number(addTagMatch[1])
      const { tagId } = request.postDataJSON() as { tagId: number }
      const tag = state.tags.find((item) => item.id === tagId)
      if (!tag) {
        await json(route, { error: 'Tag not found' }, 404)
        return
      }
      state.tagPayloads.push({ action: 'add', coinId, tagId })
      state.coins = state.coins.map((coin) => coin.id === coinId
        ? cloneCoin({
            ...coin,
            tags: coin.tags?.some((item) => item.id === tagId) ? coin.tags : [...(coin.tags ?? []), { ...tag }],
          })
        : coin)
      await json(route, { message: 'Tag attached' })
      return
    }

    const removeTagMatch = path.match(/^\/coins\/(\d+)\/tags\/(\d+)$/)
    if (removeTagMatch && method === 'DELETE') {
      const coinId = Number(removeTagMatch[1])
      const tagId = Number(removeTagMatch[2])
      state.tagPayloads.push({ action: 'remove', coinId, tagId })
      state.coins = state.coins.map((coin) => coin.id === coinId
        ? cloneCoin({ ...coin, tags: coin.tags?.filter((tag) => tag.id !== tagId) ?? [] })
        : coin)
      await json(route, { message: 'Tag detached' })
      return
    }

    const addSetMatch = path.match(/^\/sets\/(\d+)\/coins$/)
    if (addSetMatch && method === 'POST') {
      const setId = Number(addSetMatch[1])
      const { coinId } = request.postDataJSON() as { coinId: number }
      const set = state.sets.find((item) => item.id === setId)
      if (!set) {
        await json(route, { error: 'Set not found' }, 404)
        return
      }
      state.setPayloads.push({ action: 'add', coinId, setId })
      state.coins = state.coins.map((coin) => coin.id === coinId
        ? cloneCoin({
            ...coin,
            sets: coin.sets?.some((item) => item.id === setId) ? coin.sets : [...(coin.sets ?? []), { ...set }],
          })
        : coin)
      await json(route, { message: 'Coin added to set' })
      return
    }

    const removeSetMatch = path.match(/^\/sets\/(\d+)\/coins\/(\d+)$/)
    if (removeSetMatch && method === 'DELETE') {
      const setId = Number(removeSetMatch[1])
      const coinId = Number(removeSetMatch[2])
      state.setPayloads.push({ action: 'remove', coinId, setId })
      state.coins = state.coins.map((coin) => coin.id === coinId
        ? cloneCoin({ ...coin, sets: coin.sets?.filter((set) => set.id !== setId) ?? [] })
        : coin)
      await json(route, { message: 'Coin removed from set' })
      return
    }

    if (path === '/deep-identification/jobs' && method === 'POST') {
      const id = nextDeepJobId++
      const rawBody = request.postData() ?? ''
      const source = /name="coinId"/.test(rawBody) ? 'saved_coin' : 'intake'
      state.deepIdentificationJobs.push({
        id,
        notes: String(rawBody.includes('notes') ? 'submitted' : ''),
        providers: '',
        status: 'running',
        source,
      })
      await route.fulfill({
        status: 202,
        contentType: 'application/json',
        body: JSON.stringify({
          job: {
            id,
            source,
            status: 'running',
            partialSuccess: false,
            cancelRequested: false,
            lastSeq: 0,
            eventsAvailable: false,
            expiresAt: '2030-01-01T00:00:00Z',
            createdAt: '2030-01-01T00:00:00Z',
          },
        }),
      })
      return
    }

    if (path.match(/^\/deep-identification\/jobs\/\d+$/) && method === 'GET') {
      const id = Number(path.split('/').pop())
      const job = state.deepIdentificationJobs.find((item) => item.id === id)
      await json(route, {
        job: {
          id,
          source: job?.source ?? 'intake',
          status: job?.status ?? 'queued',
          partialSuccess: Boolean(job?.report && (job.report as { partialSuccess?: boolean }).partialSuccess),
          cancelRequested: false,
          lastSeq: 0,
          eventsAvailable: false,
          appliedAt: job?.appliedAt ?? null,
          appliedCoinId: job?.appliedCoinId ?? null,
          appliedDraftId: job?.appliedDraftId ?? null,
          expiresAt: '2030-01-01T00:00:00Z',
          createdAt: '2030-01-01T00:00:00Z',
        },
        report: job?.report ?? null,
        proposal: job?.proposal ? { schemaVersion: 1, fields: job.proposal } : null,
      })
      return
    }

    const proposalMatch = path.match(/^\/deep-identification\/jobs\/(\d+)\/proposal$/)
    if (proposalMatch && method === 'PATCH') {
      const id = Number(proposalMatch[1])
      const job = state.deepIdentificationJobs.find((item) => item.id === id)
      const body = JSON.parse(request.postData() ?? '{}') as { fields?: Record<string, { ownerValue?: unknown; accepted?: boolean | null }> }
      state.deepIdentificationProposalUpdates.push({ id, fields: body.fields ?? {} })
      if (job?.proposal && body.fields) {
        for (const [name, edit] of Object.entries(body.fields)) {
          const entry = job.proposal[name]
          if (!entry) continue
          if (edit.ownerValue !== undefined) {
            entry.ownerValue = edit.ownerValue
            entry.ownerEdited = true
          }
          if (edit.accepted !== undefined) entry.accepted = edit.accepted ?? null
        }
      }
      await json(route, job?.proposal ? { schemaVersion: 1, fields: job.proposal } : { schemaVersion: 1, fields: {} })
      return
    }

    const applyMatch = path.match(/^\/deep-identification\/jobs\/(\d+)\/apply$/)
    if (applyMatch && method === 'POST') {
      const id = Number(applyMatch[1])
      const job = state.deepIdentificationJobs.find((item) => item.id === id)
      const body = JSON.parse(request.postData() ?? '{}') as { target?: string }
      const target = body.target ?? 'draft'
      state.deepIdentificationApplies.push({ id, target })
      const appliedAt = '2030-01-02T00:00:00Z'
      if (job) {
        job.appliedAt = appliedAt
        if (target === 'coin') job.appliedCoinId = 8001
        else job.appliedDraftId = 8002
      }
      await json(route, {
        jobId: id,
        coinId: target === 'coin' ? 8001 : null,
        draftId: target === 'draft' ? 8002 : null,
        appliedFields: job?.proposal ? Object.keys(job.proposal).filter((name) => job.proposal![name].accepted === true) : [],
        appliedAt,
      })
      return
    }

    const cancelMatch = path.match(/^\/deep-identification\/jobs\/(\d+)\/cancel$/)
    if (cancelMatch && method === 'POST') {
      const id = Number(cancelMatch[1])
      state.deepIdentificationCancels.push(id)
      const job = state.deepIdentificationJobs.find((item) => item.id === id)
      if (job) job.status = 'cancelled'
      await json(route, {
        job: {
          id,
          source: 'intake',
          status: 'cancelled',
          partialSuccess: false,
          cancelRequested: true,
          lastSeq: 1,
          eventsAvailable: true,
          expiresAt: '2030-01-01T00:00:00Z',
          createdAt: '2030-01-01T00:00:00Z',
        },
      })
      return
    }

    const eventsMatch = path.match(/^\/deep-identification\/jobs\/(\d+)\/events$/)
    if (eventsMatch && method === 'GET') {
      const id = Number(eventsMatch[1])
      const job = state.deepIdentificationJobs.find((item) => item.id === id)
      const frames = [
        `id: 1\nevent: job_accepted\ndata: ${JSON.stringify({ seq: 1, jobId: id, type: 'job_accepted', ts: '2030-01-01T00:00:00Z', payload: { status: 'queued' } })}\n\n`,
        `id: 2\nevent: progress\ndata: ${JSON.stringify({ seq: 2, jobId: id, type: 'progress', ts: '2030-01-01T00:00:00Z', payload: { message: 'Running providers' } })}\n\n`,
      ]
      if (job && ['completed', 'partial', 'failed', 'cancelled'].includes(job.status)) {
        frames.push(`id: 3\nevent: terminal\ndata: ${JSON.stringify({ seq: 3, jobId: id, type: 'terminal', ts: '2030-01-01T00:00:00Z', payload: { status: job.status } })}\n\n`)
      }
      await route.fulfill({
        status: 200,
        contentType: 'text/event-stream',
        body: frames.join(''),
      })
      return
    }

    await json(route, {})
  })

  return state
}

export function coinFormControl(page: Page, labelText: string) {
  return page.locator('.form-group').filter({ hasText: labelText }).first().locator('input, textarea, select').first()
}

export async function expectAuthenticatedApiCall(state: WorkflowApiState, requestName: string) {
  await expect.poll(() => state.authorizedRequests).toContain(requestName)
}

function cloneCoin(coin: Coin): Coin {
  return {
    ...coin,
    storageLocation: coin.storageLocation ? { ...coin.storageLocation } : null,
    images: coin.images.map((image) => ({ ...image })),
    references: coin.references?.map((reference) => ({ ...reference })),
    tags: coin.tags?.map((tag) => ({ ...tag })),
    sets: coin.sets?.map((set) => ({ ...set })),
  }
}

function filterCoins(coins: Coin[], searchParams: URLSearchParams): Coin[] {
  const search = searchParams.get('search')?.trim().toLowerCase()
  const category = searchParams.get('category')?.trim()
  const era = searchParams.get('era')?.trim()
  const tag = searchParams.get('tag')?.trim()
  const set = searchParams.get('set')?.trim()
  const wishlist = searchParams.get('wishlist')
  const sold = searchParams.get('sold')

  return coins.filter((coin) => {
    if (wishlist === 'false' && coin.isWishlist) return false
    if (wishlist === 'true' && !coin.isWishlist) return false
    if (sold === 'false' && coin.isSold) return false
    if (sold === 'true' && !coin.isSold) return false
    if (category && coin.category !== category) return false
    if (era && coin.era !== era) return false
    if (tag && !coin.tags?.some((coinTag) => String(coinTag.id) === tag)) return false
    if (set && !coin.sets?.some((coinSet) => String(coinSet.id) === set)) return false
    if (search) {
      const haystack = [
        coin.name,
        coin.ruler,
        coin.denomination,
        coin.mint,
        coin.obverseInscription,
        coin.reverseInscription,
        coin.obverseDescription,
        coin.reverseDescription,
      ].join(' ').toLowerCase()
      if (!haystack.includes(search)) return false
    }
    return true
  })
}

async function json(route: Route, body: unknown, status = 200) {
  await route.fulfill({
    status,
    contentType: 'application/json',
    body: JSON.stringify(body),
  })
}

async function media(route: Route, state: WorkflowApiState) {
  const request = route.request()
  const url = new URL(request.url())
  state.mediaRequests.push({
    path: url.pathname,
    authorization: request.headers()['authorization'] ?? '',
    cacheControl: request.headers()['cache-control'] ?? '',
  })
  await route.fulfill({
    status: 200,
    contentType: 'image/png',
    body: Buffer.from(
      'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII=',
      'base64',
    ),
  })
}
