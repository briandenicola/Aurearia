import { describe, expect, it, vi } from 'vitest'
import { BACKGROUND_REMOVAL_ASSET_PATH, backgroundRemovalConfig, removeCoinBackground } from '@/utils/backgroundRemoval'

vi.mock('@imgly/background-removal', () => ({
  removeBackground: vi.fn().mockResolvedValue(new Blob(['processed'], { type: 'image/png' })),
}))

describe('backgroundRemoval', () => {
  it('uses same-origin quantized model assets for production background removal', () => {
    expect(backgroundRemovalConfig).toMatchObject({
      publicPath: `${window.location.origin}${BACKGROUND_REMOVAL_ASSET_PATH}`,
      model: 'isnet_quint8',
      device: 'cpu',
      proxyToWorker: false,
      output: { format: 'image/png', quality: 1 },
    })
  })

  it('uses an absolute publicPath that IMG.LY can resolve resources.json against', () => {
    expect(new URL('resources.json', backgroundRemovalConfig.publicPath).href)
      .toBe(`${window.location.origin}${BACKGROUND_REMOVAL_ASSET_PATH}resources.json`)
  })

  it('passes the shared config to imgly background removal', async () => {
    const { removeBackground } = await import('@imgly/background-removal')
    const input = new Blob(['coin'], { type: 'image/png' })

    await removeCoinBackground(input)

    expect(removeBackground).toHaveBeenCalledWith(input, backgroundRemovalConfig)
  })
})
