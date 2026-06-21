import { removeBackground, type Config, type ImageSource } from '@imgly/background-removal'

export const BACKGROUND_REMOVAL_ASSET_PATH = '/imgly-background-removal/'

export const backgroundRemovalConfig: Config = {
  publicPath: BACKGROUND_REMOVAL_ASSET_PATH,
  model: 'isnet_quint8',
  device: 'cpu',
  proxyToWorker: false,
  output: { format: 'image/png', quality: 1 },
}

export function removeCoinBackground(image: ImageSource): Promise<Blob> {
  return removeBackground(image, backgroundRemovalConfig)
}
