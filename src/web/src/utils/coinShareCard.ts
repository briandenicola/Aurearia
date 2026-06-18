import type { Coin, CoinImage } from '@/types'

const CARD_WIDTH = 1080
const CARD_HEIGHT = 1350
const CARD_PADDING = 72
const IMAGE_FRAME_Y = 96
const IMAGE_FRAME_SIZE = 560
const TITLE_START_Y = 730
const METADATA_START_Y = 910
const FOOTER_Y = 1278

const TOKEN_COLORS = {
  bgPrimary: '#0f172a',
  bgSecondary: '#1a1a2e',
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

interface TextLine {
  text: string
  y: number
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

function wrapLines(ctx: CanvasRenderingContext2D, text: string, maxWidth: number, maxLines: number): string[] {
  const words = text.split(/\s+/).filter(Boolean)
  const lines: string[] = []
  let line = ''

  for (const word of words) {
    const next = line ? `${line} ${word}` : word
    if (ctx.measureText(next).width > maxWidth && line) {
      lines.push(line)
      line = word
      if (lines.length === maxLines - 1) break
    } else {
      line = next
    }
  }
  if (line && lines.length < maxLines) lines.push(line)

  if (lines.length === maxLines && words.join(' ') !== lines.join(' ')) {
    let last = lines[lines.length - 1] ?? ''
    while (last.length > 4 && ctx.measureText(`${last}...`).width > maxWidth) {
      last = last.slice(0, -1).trimEnd()
    }
    lines[lines.length - 1] = `${last}...`
  }

  return lines
}

function drawCenteredWrappedText(
  ctx: CanvasRenderingContext2D,
  text: string,
  y: number,
  maxWidth: number,
  lineHeight: number,
  maxLines: number,
): TextLine[] {
  const lines = wrapLines(ctx, text, maxWidth, maxLines)
  const drawn: TextLine[] = []
  lines.forEach((line, index) => {
    const lineY = y + index * lineHeight
    ctx.fillText(line, CARD_WIDTH / 2, lineY)
    drawn.push({ text: line, y: lineY })
  })
  return drawn
}

function drawBackground(ctx: CanvasRenderingContext2D, width: number, height: number) {
  const gradient = ctx.createLinearGradient(0, 0, width, height)
  gradient.addColorStop(0, TOKEN_COLORS.bgCard)
  gradient.addColorStop(0.55, TOKEN_COLORS.bgPrimary)
  gradient.addColorStop(1, '#0b1020')
  ctx.fillStyle = gradient
  ctx.fillRect(0, 0, width, height)

  ctx.fillStyle = 'rgba(201, 168, 76, 0.08)'
  ctx.beginPath()
  ctx.arc(width / 2, 360, 430, 0, Math.PI * 2)
  ctx.fill()
}

function drawImageFrame(ctx: CanvasRenderingContext2D, image: HTMLImageElement | null) {
  const frameX = (CARD_WIDTH - IMAGE_FRAME_SIZE) / 2
  const radius = IMAGE_FRAME_SIZE / 2

  ctx.save()
  ctx.fillStyle = TOKEN_COLORS.bgInput
  ctx.strokeStyle = TOKEN_COLORS.borderSubtle
  ctx.lineWidth = 4
  ctx.beginPath()
  ctx.arc(CARD_WIDTH / 2, IMAGE_FRAME_Y + radius, radius, 0, Math.PI * 2)
  ctx.fill()
  ctx.stroke()
  ctx.clip()

  if (image) {
    drawContainedImage(ctx, image, frameX, IMAGE_FRAME_Y, IMAGE_FRAME_SIZE, IMAGE_FRAME_SIZE)
  } else {
    ctx.fillStyle = TOKEN_COLORS.textMuted
    ctx.font = '600 36px Inter, sans-serif'
    ctx.textAlign = 'center'
    ctx.textBaseline = 'middle'
    ctx.fillText('No coin image', CARD_WIDTH / 2, IMAGE_FRAME_Y + radius)
  }

  ctx.restore()

  ctx.strokeStyle = TOKEN_COLORS.accentGold
  ctx.lineWidth = 5
  ctx.beginPath()
  ctx.arc(CARD_WIDTH / 2, IMAGE_FRAME_Y + radius, radius + 6, 0, Math.PI * 2)
  ctx.stroke()
}

function drawMetadataGrid(ctx: CanvasRenderingContext2D, fields: CoinShareCardField[]) {
  const columns = 2
  const columnGap = 48
  const columnWidth = (CARD_WIDTH - CARD_PADDING * 2 - columnGap) / columns
  const rowHeight = 96

  ctx.textAlign = 'left'
  ctx.textBaseline = 'alphabetic'

  fields.slice(0, 6).forEach((field, index) => {
    const column = index % columns
    const row = Math.floor(index / columns)
    const x = CARD_PADDING + column * (columnWidth + columnGap)
    const y = METADATA_START_Y + row * rowHeight

    ctx.fillStyle = TOKEN_COLORS.textMuted
    ctx.font = '600 22px Inter, sans-serif'
    ctx.fillText(field.label.toUpperCase(), x, y)

    ctx.fillStyle = TOKEN_COLORS.textSecondary
    ctx.font = '400 30px Inter, sans-serif'
    const valueLines = wrapLines(ctx, field.value, columnWidth, 2)
    valueLines.forEach((line, lineIndex) => {
      ctx.fillText(line, x, y + 38 + lineIndex * 34)
    })
  })
}

function drawMetadata(ctx: CanvasRenderingContext2D, metadata: CoinShareCardMetadata, appName: string) {
  ctx.textAlign = 'center'
  ctx.textBaseline = 'alphabetic'
  ctx.fillStyle = TOKEN_COLORS.textPrimary
  ctx.font = '600 54px Cinzel, serif'
  const titleLines = drawCenteredWrappedText(ctx, metadata.title, TITLE_START_Y, 900, 58, 2)

  let categoryY = TITLE_START_Y + 126
  if (titleLines.length === 1) {
    categoryY = TITLE_START_Y + 80
  }

  if (metadata.category) {
    ctx.font = '600 26px Inter, sans-serif'
    ctx.fillStyle = TOKEN_COLORS.accentGold
    ctx.fillText(metadata.category.toUpperCase(), CARD_WIDTH / 2, categoryY)
  }

  drawMetadataGrid(ctx, metadata.fields)

  ctx.fillStyle = TOKEN_COLORS.accentBronze
  ctx.font = '600 28px Cinzel, serif'
  ctx.textAlign = 'center'
  ctx.fillText(appName, CARD_WIDTH / 2, FOOTER_Y)
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
