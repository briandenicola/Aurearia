import { removeBackground, type Config, type ImageSource } from '@imgly/background-removal'

export const BACKGROUND_REMOVAL_ASSET_PATH = '/imgly-background-removal/'

export function backgroundRemovalPublicPath(): string {
  return new URL(BACKGROUND_REMOVAL_ASSET_PATH, window.location.origin).href
}

export const backgroundRemovalConfig: Config = {
  publicPath: backgroundRemovalPublicPath(),
  model: 'isnet_quint8',
  device: 'cpu',
  proxyToWorker: false,
  output: { format: 'image/png', quality: 1 },
}

export function removeCoinBackground(image: ImageSource): Promise<Blob> {
  return removeBackground(image, backgroundRemovalConfig)
}
