import type { Coin, CoinImage } from '@/types'

const CARD_WIDTH = 1080
const CARD_HEIGHT = 1350
const IMAGE_FRAME_SIZE = 760
const TEXT_START_Y = 940

const TOKEN_COLORS = {
  bgPrimary: '#0f172a',
  bgCard: '#16213e',
  bgInput: '#1e2a4a',
  accentGold: '#c9a84c',
  accentBronze: '#b08d57',
  textPrimary: '#e8e0d0',
  textSecondary: '#a09880',
  textMuted: '#706858',
  borderSubtle: 'rgba(201, 168, 76, 0.28)',
}

export interface CoinShareCardField {
  label: string
  value: string
}

export interface CoinShareCardMetadata {
  title: string
  category: string
  fields: CoinShareCardField[]
}

export interface CoinShareCardInput {
  coin: Coin
  imageUrl: string | null
  appName: string
}

export interface CoinShareCardRenderOptions {
  width?: number
  height?: number
}

function cleanText(value: string | null | undefined): string {
  return value?.trim() ?? ''
}

function addField(fields: CoinShareCardField[], label: string, value: string | null | undefined) {
  const clean = cleanText(value)
  if (clean) {
    fields.push({ label, value: clean })
  }
}

export function getShareCardMetadata(coin: Coin): CoinShareCardMetadata {
  const fields: CoinShareCardField[] = []
  addField(fields, 'Ruler', coin.ruler)
  addField(fields, 'Denomination', coin.denomination)
  addField(fields, 'Era', coin.era)
  addField(fields, 'Mint', coin.mint)
  addField(fields, 'Material', coin.material)
  addField(fields, 'Grade', coin.grade)

  return {
    title: cleanText(coin.name) || 'Untitled Coin',
    category: cleanText(coin.category),
    fields,
  }
}

function imagePath(image: CoinImage): string {
  const path = image.filePath.trim()
  if (path.startsWith('/uploads/')) return path
  return `/uploads/${path.replace(/^\/+/, '')}`
}

export function getPreferredShareImage(coin: Coin): string | null {
  const obverse = coin.images?.find((image) => image.imageType === 'obverse')
  if (obverse) return imagePath(obverse)

  const primary = coin.images?.find((image) => image.isPrimary)
  if (primary) return imagePath(primary)

  const first = coin.images?.[0]
  return first ? imagePath(first) : null
}

export function getShareCardFilename(coin: Coin): string {
  const base = cleanText(coin.name)
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 80)

  return `${base || 'coin'}-share-card.png`
}

function loadImage(src: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const image = new Image()
    image.onload = () => resolve(image)
    image.onerror = () => reject(new Error('Unable to load coin image for share card.'))
    image.src = src
  })
}

function drawContainedImage(
  ctx: CanvasRenderingContext2D,
  image: HTMLImageElement,
  x: number,
  y: number,
  maxWidth: number,
  maxHeight: number,
) {
  const width = image.naturalWidth || image.width
  const height = image.naturalHeight || image.height
  if (!width || !height) return

  const scale = Math.min(maxWidth / width, maxHeight / height)
  const drawWidth = width * scale
  const drawHeight = height * scale
  ctx.drawImage(image, x + (maxWidth - drawWidth) / 2, y + (maxHeight - drawHeight) / 2, drawWidth, drawHeight)
}

function drawCenteredText(ctx: CanvasRenderingContext2D, text: string, y: number, maxWidth: number) {
  const words = text.split(/\s+/).filter(Boolean)
  const lines: string[] = []
  let line = ''

  for (const word of words) {
    const next = line ? `${line} ${word}` : word
    if (ctx.measureText(next).width > maxWidth && line) {
      lines.push(line)
      line = word
    } else {
      line = next
    }
  }
  if (line) lines.push(line)

  lines.slice(0, 2).forEach((part, index) => {
    ctx.fillText(part, CARD_WIDTH / 2, y + index * 54)
  })
}

function drawBackground(ctx: CanvasRenderingContext2D, width: number, height: number) {
  const gradient = ctx.createLinearGradient(0, 0, width, height)
  gradient.addColorStop(0, TOKEN_COLORS.bgCard)
  gradient.addColorStop(0.58, TOKEN_COLORS.bgPrimary)
  gradient.addColorStop(1, '#0b1020')
  ctx.fillStyle = gradient
  ctx.fillRect(0, 0, width, height)

  ctx.fillStyle = 'rgba(201, 168, 76, 0.08)'
  ctx.beginPath()
  ctx.arc(width / 2, 420, 470, 0, Math.PI * 2)
  ctx.fill()
}

function drawImageFrame(ctx: CanvasRenderingContext2D, image: HTMLImageElement | null) {
  const frameX = (CARD_WIDTH - IMAGE_FRAME_SIZE) / 2
  const frameY = 130
  const radius = IMAGE_FRAME_SIZE / 2

  ctx.save()
  ctx.fillStyle = TOKEN_COLORS.bgInput
  ctx.strokeStyle = TOKEN_COLORS.borderSubtle
  ctx.lineWidth = 4
  ctx.beginPath()
  ctx.arc(CARD_WIDTH / 2, frameY + radius, radius, 0, Math.PI * 2)
  ctx.fill()
  ctx.stroke()
  ctx.clip()

  if (image) {
    drawContainedImage(ctx, image, frameX, frameY, IMAGE_FRAME_SIZE, IMAGE_FRAME_SIZE)
  } else {
    ctx.fillStyle = TOKEN_COLORS.textMuted
    ctx.font = '600 42px Inter, sans-serif'
    ctx.textAlign = 'center'
    ctx.textBaseline = 'middle'
    ctx.fillText('No coin image', CARD_WIDTH / 2, frameY + radius)
  }

  ctx.restore()

  ctx.strokeStyle = TOKEN_COLORS.accentGold
  ctx.lineWidth = 6
  ctx.beginPath()
  ctx.arc(CARD_WIDTH / 2, frameY + radius, radius + 5, 0, Math.PI * 2)
  ctx.stroke()
}

function drawMetadata(ctx: CanvasRenderingContext2D, metadata: CoinShareCardMetadata, appName: string) {
  ctx.textAlign = 'center'
  ctx.textBaseline = 'alphabetic'
  ctx.fillStyle = TOKEN_COLORS.textPrimary
  ctx.font = '600 48px Cinzel, serif'
  drawCenteredText(ctx, metadata.title, TEXT_START_Y, 860)

  let y = TEXT_START_Y + 150
  ctx.font = '600 26px Inter, sans-serif'
  ctx.fillStyle = TOKEN_COLORS.accentGold
  if (metadata.category) {
    ctx.fillText(metadata.category.toUpperCase(), CARD_WIDTH / 2, y)
    y += 58
  }

  ctx.font = '400 32px Inter, sans-serif'
  for (const field of metadata.fields.slice(0, 6)) {
    ctx.fillStyle = TOKEN_COLORS.textMuted
    ctx.fillText(field.label.toUpperCase(), CARD_WIDTH / 2, y)
    y += 38
    ctx.fillStyle = TOKEN_COLORS.textSecondary
    ctx.fillText(field.value, CARD_WIDTH / 2, y)
    y += 54
  }

  ctx.fillStyle = TOKEN_COLORS.accentBronze
  ctx.font = '600 28px Cinzel, serif'
  ctx.fillText(appName, CARD_WIDTH / 2, CARD_HEIGHT - 72)
}

function canvasToPngBlob(canvas: HTMLCanvasElement): Promise<Blob> {
  return new Promise((resolve, reject) => {
    canvas.toBlob((blob) => {
      if (blob) {
        resolve(blob)
      } else {
        reject(new Error('Unable to generate share card image.'))
      }
    }, 'image/png')
  })
}

export async function renderCoinShareCard(
  input: CoinShareCardInput,
  options: CoinShareCardRenderOptions = {},
): Promise<Blob> {
  const width = options.width ?? CARD_WIDTH
  const height = options.height ?? CARD_HEIGHT
  const canvas = document.createElement('canvas')
  canvas.width = width
  canvas.height = height

  const ctx = canvas.getContext('2d')
  if (!ctx) {
    throw new Error('Canvas rendering is not available in this browser.')
  }

  const metadata = getShareCardMetadata(input.coin)
  const image = input.imageUrl ? await loadImage(input.imageUrl) : null

  drawBackground(ctx, width, height)
  drawImageFrame(ctx, image)
  drawMetadata(ctx, metadata, input.appName)

  return canvasToPngBlob(canvas)
}
