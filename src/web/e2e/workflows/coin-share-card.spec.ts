import { readFile } from 'node:fs/promises'
import { expect, test, type Download } from '@playwright/test'
import { buildRomanDenariusCore } from '../../src/test/fixtures'
import { installAuthenticatedSession, installWorkflowApiMocks } from '../fixtures/workflow'

declare global {
  interface Window {
    shareCardText: string[]
    sharedAttributionFiles: { name: string; type: string; bytes: number[] }[]
  }

}

const originalCardBytes = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII=',
  'base64',
)

function coinWithOriginals() {
  const coin = buildRomanDenariusCore()
  return {
    ...coin,
    images: [...coin.images, ...[71, 72].map(id => ({
      id, coinId: coin.id, filePath: `coin-${coin.id}/original-card-${id}.png`,
      imageType: 'card' as const, isPrimary: false, createdAt: '',
    }))],
  }
}

for (const viewport of [{ width: 1440, height: 900 }, { width: 390, height: 844 }]) {
  test(`requires opt-in and supplies separate original native-share files at ${viewport.width}px`, async ({ page }) => {
    await page.setViewportSize(viewport)
    await installAuthenticatedSession(page)
    await page.addInitScript(() => {
      window.sharedAttributionFiles = []
      Object.defineProperty(navigator, 'canShare', { configurable: true, value: () => true })
      Object.defineProperty(navigator, 'share', { configurable: true, value: async (data: ShareData) => {
        window.sharedAttributionFiles = await Promise.all((data.files ?? []).map(async file => ({
          name: file.name,
          type: file.type,
          bytes: file.name.includes('original-attribution') ? [...new Uint8Array(await file.arrayBuffer())] : [],
        })))
      } })
    })
    const coin = coinWithOriginals()
    const state = await installWorkflowApiMocks(page, [coin])
    await page.goto(`/coin/${coin.id}`)

    await page.getByRole('button', { name: 'Share', exact: true }).click()
    await page.getByRole('button', { name: 'Generated card only', exact: true }).click()
    await expect.poll(() => page.evaluate(() => window.sharedAttributionFiles.length)).toBe(1)
    expect(state.mediaRequests.some(request => request.path.includes('original-card'))).toBe(false)

    await page.getByRole('button', { name: 'Share', exact: true }).click()
    await page.getByRole('button', { name: 'Include originals', exact: true }).click()
    await expect.poll(() => page.evaluate(() => window.sharedAttributionFiles.length)).toBe(3)
    const files = await page.evaluate(() => window.sharedAttributionFiles)
    expect(files.map(file => file.name)).toEqual([
      'trajan-denarius-core-share-card.png',
      'trajan-denarius-core-original-attribution-71.png',
      'trajan-denarius-core-original-attribution-72.png',
    ])
    for (const file of files.slice(1)) {
      expect(file.type).toBe('image/png')
      expect(Buffer.from(file.bytes)).toEqual(originalCardBytes)
    }
    const originalRequests = state.mediaRequests.filter(request => request.path.includes('original-card'))
    expect(originalRequests).toHaveLength(2)
    expect(originalRequests.every(request => request.authorization === 'Bearer workflow-access-token')).toBe(true)
  })

  test(`downloads every opted-in original as a separate file at ${viewport.width}px`, async ({ page }, testInfo) => {
    await page.setViewportSize(viewport)
    await installAuthenticatedSession(page)
    await page.addInitScript(() => {
      Object.defineProperty(navigator, 'canShare', { configurable: true, value: () => false })
    })
    const coin = coinWithOriginals()
    await installWorkflowApiMocks(page, [coin])
    await page.goto(`/coin/${coin.id}`)
    const downloads: Download[] = []
    page.on('download', download => downloads.push(download))

    await page.getByRole('button', { name: 'Share', exact: true }).click()
    await page.getByRole('button', { name: 'Include originals', exact: true }).click()
    await expect(page.getByText('Allow multiple downloads', { exact: false })).toBeVisible()
    await page.getByRole('button', { name: 'OK', exact: true }).click()
    await expect.poll(() => downloads.length).toBe(3)

    expect(downloads.map(download => download.suggestedFilename())).toEqual([
      'trajan-denarius-core-share-card.png',
      'trajan-denarius-core-original-attribution-71.png',
      'trajan-denarius-core-original-attribution-72.png',
    ])
    for (const download of downloads.slice(1)) {
      const output = testInfo.outputPath(download.suggestedFilename())
      await download.saveAs(output)
      expect(await readFile(output)).toEqual(originalCardBytes)
    }
  })

  test(`downloads the collector-ticket share card at ${viewport.width}px`, async ({ page }, testInfo) => {
    await page.setViewportSize(viewport)
    await installAuthenticatedSession(page)
    await page.addInitScript(() => {
      Object.defineProperty(navigator, 'canShare', { configurable: true, value: () => false })
      window.shareCardText = []
      const fillText = CanvasRenderingContext2D.prototype.fillText
      CanvasRenderingContext2D.prototype.fillText = function (text, x, y, maxWidth) {
        window.shareCardText.push(text)
        if (maxWidth === undefined) return fillText.call(this, text, x, y)
        return fillText.call(this, text, x, y, maxWidth)
      }
    })
    const coin = buildRomanDenariusCore({
      dateRange: 'AD 98-117',
      notes: 'PRIVATE NOTE NOT FOR SHARING',
      purchaseLocation: 'PRIVATE DEALER',
    })
    await installWorkflowApiMocks(page, [coin])
    await page.goto(`/coin/${coin.id}`)

    const downloadPromise = page.waitForEvent('download')
    await page.getByRole('button', { name: 'Share', exact: true }).click()
    const download = await downloadPromise
    expect(download.suggestedFilename()).toBe('trajan-denarius-core-share-card.png')
    const output = testInfo.outputPath('collector-ticket.png')
    await download.saveAs(output)

    const rendered = await page.evaluate(async (base64) => {
      const image = new Image()
      image.src = `data:image/png;base64,${base64}`
      await image.decode()
      const canvas = document.createElement('canvas')
      canvas.width = image.naturalWidth
      canvas.height = image.naturalHeight
      const ctx = canvas.getContext('2d')
      if (!ctx) throw new Error('Canvas unavailable')
      ctx.drawImage(image, 0, 0)
      return {
        width: canvas.width,
        height: canvas.height,
        dark: [...ctx.getImageData(0, 0, 1, 1).data],
        paper: [...ctx.getImageData(10, 700, 1, 1).data],
        text: window.shareCardText.join(' '),
      }
    }, (await readFile(output)).toString('base64'))

    expect(rendered.width).toBe(1080)
    expect(rendered.height).toBeGreaterThanOrEqual(1440)
    expect(rendered.dark).toEqual([23, 25, 27, 255])
    expect(rendered.paper[0]).toBeGreaterThan(230)
    expect(rendered.paper[0]! - rendered.paper[2]!).toBeGreaterThan(20)
    expect(rendered.text).toContain(`${coin.name} Trajan, AD 98-117`)
    expect(rendered.text).toContain('Obv. IMP TRAIANO')
    expect(rendered.text).toContain('Rev. SPQR OPTIMO PRINCIPI.')
    expect(rendered.text).toContain('References: RIC II Trajan 147')
    expect(rendered.text).not.toContain('PRIVATE NOTE')
    expect(rendered.text).not.toContain('PRIVATE DEALER')
    await testInfo.attach('Collector ticket PNG', { path: output, contentType: 'image/png' })
  })
}
