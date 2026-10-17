import { readFile } from 'node:fs/promises'
import { expect, test } from '@playwright/test'
import { buildRomanDenariusCore } from '../../src/test/fixtures'
import { installAuthenticatedSession, installWorkflowApiMocks } from '../fixtures/workflow'

declare global {
  interface Window {
    shareCardText: string[]
  }
}

for (const viewport of [{ width: 1440, height: 900 }, { width: 390, height: 844 }]) {
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
    expect(rendered.text).toContain('Trajan, AD 98-117')
    expect(rendered.text).toContain('Obv. IMP TRAIANO')
    expect(rendered.text).toContain('Rev. SPQR OPTIMO PRINCIPI.')
    expect(rendered.text).toContain('References: RIC II Trajan 147')
    expect(rendered.text).not.toContain('PRIVATE NOTE')
    expect(rendered.text).not.toContain('PRIVATE DEALER')
    await testInfo.attach('Collector ticket PNG', { path: output, contentType: 'image/png' })
  })
}
