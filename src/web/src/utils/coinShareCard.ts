import type { Coin, CoinImage } from '@/types'
import { isPrivateUploadPath, privateMediaObjectUrl } from '@/utils/media'

const CARD_WIDTH = 1080
const CARD_HEIGHT = 1440
const TICKET_Y = 672
const CARD_PADDING = 64

const CARD_COLORS = {
  background: '#17191b',
  paperStart: '#efe1c4',
  paperMiddle: '#f6ebd4',
  paperEnd: '#ecddbd',
  ink: '#322a20',
  brand: '#c8b68c',
  imageLabel: '#9e9c95',
  footer: '#83745b',
}

export interface CoinShareCardMetadata {
  title: string
  details: string[]
  obverse: string
  reverse: string
  references: string
}

export interface CoinShareCardInput {
  coin: Coin
  imageUrl: string | null
  imageUrls?: string[]
  appName: string
  context?: CoinShareCardContext
}

export interface CoinShareCardRenderOptions {
  width?: number
  height?: number
}

export interface CoinShareCardContext {
  heading?: string
  summary?: string
}

interface TicketBlock {
  lines: string[]
  font: string
  lineHeight: number
  y: number
}

function cleanText(value: string | null | undefined): string {
  return value?.trim() ?? ''
}

function sentence(value: string): string {
  return value && !/[.!?]$/.test(value) ? `${value}.` : value
}

export function getShareCardMetadata(coin: Coin): CoinShareCardMetadata {
  const title = cleanText(coin.name) || 'Untitled Coin'
  const date = cleanText(coin.dateRange)
  const mint = cleanText(coin.mint)
  const issue = [cleanText(coin.material), cleanText(coin.denomination)].filter(Boolean).join(' ')
  const dimensions: string[] = []
  if (coin.weightGrams != null && Number.isFinite(coin.weightGrams) && coin.weightGrams > 0) {
    dimensions.push(`${coin.weightGrams.toFixed(2)} g`)
  }
  if (coin.diameterMm != null && Number.isFinite(coin.diameterMm) && coin.diameterMm > 0) {
    dimensions.push(`${coin.diameterMm} mm`)
  }
  const references = (coin.references ?? [])
    .map((reference) => [reference.catalog, reference.volume, reference.number].map(cleanText).filter(Boolean).join(' '))
    .filter(Boolean)

  return {
    title,
    details: [
      [cleanText(coin.ruler), date].filter(Boolean).join(', '),
      [mint ? sentence(/\bmint$/i.test(mint) ? mint : `${mint} mint`) : '', sentence(issue)].filter(Boolean).join(' '),
      sentence(dimensions.join('; ')),
    ].filter(Boolean),
    obverse: [sentence(cleanText(coin.obverseInscription)), cleanText(coin.obverseDescription)].filter(Boolean).join(' '),
    reverse: [sentence(cleanText(coin.reverseInscription)), cleanText(coin.reverseDescription)].filter(Boolean).join(' '),
    references: [...new Set(references)].join('; ') || cleanText(coin.referenceText),
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

  const primary = coin.images?.find((image) => image.isPrimary && image.imageType !== 'card')
  if (primary) return imagePath(primary)

  const first = coin.images?.find((image) => image.imageType !== 'card')
  return first ? imagePath(first) : null
}

export function getShareImageUrls(coin: Coin): string[] {
  const images: string[] = []
  const obverse = coin.images?.find((image) => image.imageType === 'obverse')
  const reverse = coin.images?.find((image) => image.imageType === 'reverse')

  if (obverse) images.push(imagePath(obverse))
  if (reverse && reverse.id !== obverse?.id) images.push(imagePath(reverse))

  if (images.length > 0) return images

  const fallback = getPreferredShareImage(coin)
  return fallback ? [fallback] : []
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

function wrapLines(ctx: CanvasRenderingContext2D, text: string, maxWidth: number): string[] {
  const words = text.split(/\s+/).filter(Boolean)
  const lines: string[] = []
  let line = ''

  for (const word of words) {
    if (line && ctx.measureText(`${line} ${word}`).width > maxWidth) {
      lines.push(line)
      line = ''
    }
    if (line) {
      line += ` ${word}`
      continue
    }
    for (const character of word) {
      if (line && ctx.measureText(line + character).width > maxWidth) {
        lines.push(line)
        line = ''
      }
      line += character
    }
  }
  if (line) lines.push(line)

  return lines
}

function plainText(value: string | null | undefined): string {
  return cleanText(value)
    .replace(/```[\s\S]*?```/g, ' ')
    .replace(/`([^`]+)`/g, '$1')
    .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')
    .replace(/[*_~>#-]+/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
}

function layoutTicket(
  ctx: CanvasRenderingContext2D,
  metadata: CoinShareCardMetadata,
  context?: CoinShareCardContext,
) {
  const blocks: TicketBlock[] = []
  let y = TICKET_Y + 56
  function add(text: string, font: string, lineHeight: number, gap: number) {
    if (!text) return
    ctx.font = font
    const lines = wrapLines(ctx, text, CARD_WIDTH - CARD_PADDING * 2)
    blocks.push({ lines, font, lineHeight, y })
    y += lines.length * lineHeight + gap
  }

  add(metadata.title, '400 52px Georgia, serif', 64, 16)
  metadata.details.forEach((detail, index) => {
    add(detail, '400 32px Georgia, serif', 44, index === metadata.details.length - 1 ? 22 : 0)
  })
  add(metadata.obverse ? `Obv. ${metadata.obverse}` : '', '400 32px Georgia, serif', 44, 26)
  add(metadata.reverse ? `Rev. ${metadata.reverse}` : '', '400 32px Georgia, serif', 44, 26)
  add(metadata.references ? `References: ${metadata.references}` : '', '400 28px Georgia, serif', 38, 26)
  add(cleanText(context?.heading), '600 28px Georgia, serif', 38, 10)
  add(plainText(context?.summary), '400 28px Georgia, serif', 38, 0)
  return { blocks, height: Math.max(CARD_HEIGHT, y + 120) }
}

function drawImageFrame(
  ctx: CanvasRenderingContext2D,
  images: HTMLImageElement[],
  imageUrls: string[],
  coin: Coin,
  appName: string,
) {
  const slots: (HTMLImageElement | null)[] = [null, null]
  const labels = ['OBVERSE', 'REVERSE']
  images.forEach((image, index) => {
    const type = coin.images?.find((candidate) => imagePath(candidate) === imageUrls[index])?.imageType
    const slot = type === 'reverse' ? 1 : type === 'obverse' ? 0 : index
    slots[slot] = image
    if (type && type !== 'obverse' && type !== 'reverse') labels[slot] = 'COIN'
  })

  ctx.textBaseline = 'top'
  ctx.textAlign = 'center'
  ctx.fillStyle = CARD_COLORS.brand
  ctx.font = '400 24px Georgia, serif'
  const brand = cleanText(appName.split(' - ')[0]) || 'Aurearia'
  ctx.fillText(brand.toUpperCase(), CARD_WIDTH / 2, 44)

  const gap = 48
  const panelWidth = (CARD_WIDTH - CARD_PADDING * 2 - gap) / 2
  slots.forEach((image, index) => {
    const x = CARD_PADDING + index * (panelWidth + gap)
    if (image) {
      drawContainedImage(ctx, image, x, 128, panelWidth, 432)
    } else {
      ctx.fillStyle = CARD_COLORS.imageLabel
      ctx.font = '400 24px Georgia, serif'
      ctx.fillText(index === 0 ? 'No obverse image' : 'No reverse image', x + panelWidth / 2, 332)
    }
    ctx.fillStyle = CARD_COLORS.imageLabel
    ctx.font = '400 18px Arial, sans-serif'
    ctx.fillText(labels[index] ?? 'COIN', x + panelWidth / 2, 602)
  })
}

function drawTicket(ctx: CanvasRenderingContext2D, blocks: TicketBlock[], height: number, appName: string) {
  const paper = ctx.createLinearGradient(0, TICKET_Y, CARD_WIDTH, height)
  paper.addColorStop(0, CARD_COLORS.paperStart)
  paper.addColorStop(0.55, CARD_COLORS.paperMiddle)
  paper.addColorStop(1, CARD_COLORS.paperEnd)
  ctx.fillStyle = paper
  ctx.fillRect(0, TICKET_Y, CARD_WIDTH, height - TICKET_Y)
  ctx.fillStyle = CARD_COLORS.ink
  ctx.textAlign = 'left'
  ctx.textBaseline = 'top'
  for (const block of blocks) {
    ctx.font = block.font
    block.lines.forEach((line, index) => {
      ctx.fillText(line, CARD_PADDING, block.y + index * block.lineHeight)
    })
  }
  ctx.fillStyle = CARD_COLORS.footer
  ctx.font = '400 16px Arial, sans-serif'
  ctx.fillText(appName, CARD_PADDING, height - 54)
  ctx.textAlign = 'right'
  ctx.fillText('ATTRIBUTION', CARD_WIDTH - CARD_PADDING, height - 54)
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
  const canvas = document.createElement('canvas')

  const ctx = canvas.getContext('2d')
  if (!ctx) {
    throw new Error('Canvas rendering is not available in this browser.')
  }

  const metadata = getShareCardMetadata(input.coin)
  const ticket = layoutTicket(ctx, metadata, input.context)
  canvas.width = width
  canvas.height = options.height ?? Math.round(ticket.height * width / CARD_WIDTH)
  const imageUrls = input.imageUrls ?? (input.imageUrl ? [input.imageUrl] : [])
  const objectUrls: string[] = []

  try {
    const resolvedImageUrls = await Promise.all(imageUrls.slice(0, 2).map(async (imageUrl) => {
      if (!isPrivateUploadPath(imageUrl)) return imageUrl
      const objectUrl = await privateMediaObjectUrl(imageUrl)
      objectUrls.push(objectUrl)
      return objectUrl
    }))
    const images = await Promise.all(resolvedImageUrls.map((imageUrl) => loadImage(imageUrl)))

    ctx.scale(canvas.width / CARD_WIDTH, canvas.height / ticket.height)
    ctx.fillStyle = CARD_COLORS.background
    ctx.fillRect(0, 0, CARD_WIDTH, TICKET_Y)
    drawImageFrame(ctx, images, imageUrls, input.coin, input.appName)
    drawTicket(ctx, ticket.blocks, ticket.height, input.appName)

    return await canvasToPngBlob(canvas)
  } finally {
    objectUrls.forEach((objectUrl) => URL.revokeObjectURL(objectUrl))
  }
}
