import { ref, watch, nextTick, onMounted, onUnmounted, type Ref } from 'vue'
import { removeBackground as removeBg } from '@imgly/background-removal'
import { proxyImage, uploadImage, createCoin, getCoins, getCoin } from '@/api/client'
import type { Coin } from '@/types'

export type Step = 'preview' | 'removing' | 'crop' | 'done'

export function useImageProcessor(
  cropCanvas: Ref<HTMLCanvasElement | null>,
  resultCanvas: Ref<HTMLCanvasElement | null>,
  options: { coinId?: number },
) {
  // Input state
  const sourceImage = ref<string | null>(null)
  const urlLoading = ref(false)
  const inputError = ref('')

  // Processing state
  const step = ref<Step>('preview')
  const processedBlob = ref<Blob | null>(null)
  const processedImage = ref<HTMLImageElement | null>(null)

  // Crop state
  const cropPadding = ref(10)
  const cropRect = ref({ x: 0, y: 0, w: 0, h: 0 })
  const cropDragging = ref(false)
  const cropDragType = ref<'move' | 'nw' | 'ne' | 'sw' | 'se' | null>(null)
  const cropDragStart = ref({ x: 0, y: 0, rx: 0, ry: 0, rw: 0, rh: 0 })
  let canvasScale = 1

  // Save state
  const saveTab = ref<'existing' | 'new' | 'download'>('existing')
  const saveImageType = ref('obverse')
  const saving = ref(false)
  const saveMsg = ref('')
  const saveError = ref(false)

  // Existing coin selection
  const coinSearch = ref('')
  const coinOptions = ref<Coin[]>([])
  const coinsLoading = ref(false)
  const selectedCoinId = ref<number | null>(null)
  let searchTimeout: ReturnType<typeof setTimeout> | null = null

  // New coin
  const newCoinName = ref('')

  // Pre-select coin if coinId provided
  onMounted(async () => {
    if (options.coinId) {
      try {
        const res = await getCoin(options.coinId)
        coinOptions.value = [res.data]
        selectedCoinId.value = res.data.id
      } catch { /* ignore */ }
    }
  })

  // --- Input Methods ---

  function loadImageFromFile(file: File) {
    inputError.value = ''
    if (!file.type.startsWith('image/')) {
      inputError.value = 'Please select an image file'
      return
    }
    const reader = new FileReader()
    reader.onload = (e) => {
      sourceImage.value = e.target?.result as string
      step.value = 'preview'
    }
    reader.readAsDataURL(file)
  }

  async function handleUrlLoad(url: string) {
    if (!url) return
    inputError.value = ''
    urlLoading.value = true
    try {
      const res = await proxyImage(url)
      const blob = res.data as Blob
      const reader = new FileReader()
      reader.onload = (e) => {
        sourceImage.value = e.target?.result as string
        step.value = 'preview'
      }
      reader.readAsDataURL(blob)
    } catch {
      inputError.value = 'Failed to fetch image from URL'
    } finally {
      urlLoading.value = false
    }
  }

  // --- Background Removal ---

  async function removeBackground() {
    if (!sourceImage.value) return
    step.value = 'removing'

    try {
      const response = await fetch(sourceImage.value)
      const srcBlob = await response.blob()
      const result = await removeBg(srcBlob, {
        output: { format: 'image/png', quality: 1 },
      })
      processedBlob.value = result
      const img = new Image()
      img.onload = () => {
        processedImage.value = img
        step.value = 'crop'
        nextTick(() => {
          autoCrop()
        })
      }
      img.src = URL.createObjectURL(result)
    } catch (err) {
      console.error('Background removal failed:', err)
      inputError.value = 'Background removal failed. Please try again.'
      step.value = 'preview'
    }
  }

  // --- Crop Logic ---

  function autoCrop() {
    if (!processedImage.value || !cropCanvas.value) return

    const img = processedImage.value
    const offscreen = document.createElement('canvas')
    offscreen.width = img.naturalWidth
    offscreen.height = img.naturalHeight
    const ctx = offscreen.getContext('2d')!
    ctx.drawImage(img, 0, 0)

    const data = ctx.getImageData(0, 0, offscreen.width, offscreen.height).data
    let minX = offscreen.width, minY = offscreen.height, maxX = 0, maxY = 0

    for (let y = 0; y < offscreen.height; y++) {
      for (let x = 0; x < offscreen.width; x++) {
        const alpha = data[(y * offscreen.width + x) * 4 + 3] ?? 0
        if (alpha > 10) {
          if (x < minX) minX = x
          if (x > maxX) maxX = x
          if (y < minY) minY = y
          if (y > maxY) maxY = y
        }
      }
    }

    if (maxX <= minX || maxY <= minY) {
      cropRect.value = { x: 0, y: 0, w: img.naturalWidth, h: img.naturalHeight }
    } else {
      const pad = cropPadding.value
      cropRect.value = {
        x: Math.max(0, minX - pad),
        y: Math.max(0, minY - pad),
        w: Math.min(img.naturalWidth - Math.max(0, minX - pad), maxX - minX + 1 + pad * 2),
        h: Math.min(img.naturalHeight - Math.max(0, minY - pad), maxY - minY + 1 + pad * 2),
      }
    }

    drawCropCanvas()
    drawResultCanvas()
  }

  function resetCrop() {
    if (!processedImage.value) return
    cropRect.value = {
      x: 0, y: 0,
      w: processedImage.value.naturalWidth,
      h: processedImage.value.naturalHeight,
    }
    drawCropCanvas()
    drawResultCanvas()
  }

  function drawCropCanvas() {
    const canvas = cropCanvas.value
    const img = processedImage.value
    if (!canvas || !img) return

    const maxW = Math.min(500, canvas.parentElement?.clientWidth || 500)
    canvasScale = maxW / img.naturalWidth
    const dispH = img.naturalHeight * canvasScale

    canvas.width = maxW
    canvas.height = dispH
    canvas.style.width = maxW + 'px'
    canvas.style.height = dispH + 'px'

    const ctx = canvas.getContext('2d')!

    drawCheckerboard(ctx, maxW, dispH)
    ctx.drawImage(img, 0, 0, maxW, dispH)

    // Dim outside crop area
    const r = cropRect.value
    const sx = r.x * canvasScale
    const sy = r.y * canvasScale
    const sw = r.w * canvasScale
    const sh = r.h * canvasScale

    ctx.fillStyle = 'rgba(0, 0, 0, 0.5)'
    ctx.fillRect(0, 0, maxW, sy)
    ctx.fillRect(0, sy + sh, maxW, dispH - sy - sh)
    ctx.fillRect(0, sy, sx, sh)
    ctx.fillRect(sx + sw, sy, maxW - sx - sw, sh)

    // Crop border
    ctx.strokeStyle = '#c9a84c'
    ctx.lineWidth = 2
    ctx.strokeRect(sx, sy, sw, sh)

    // Corner handles
    const handleSize = 8
    ctx.fillStyle = '#c9a84c'
    for (const [hx, hy] of [[sx, sy], [sx + sw, sy], [sx, sy + sh], [sx + sw, sy + sh]] as const) {
      ctx.fillRect(hx! - handleSize / 2, hy! - handleSize / 2, handleSize, handleSize)
    }
  }

  function drawResultCanvas() {
    const canvas = resultCanvas.value
    const img = processedImage.value
    if (!canvas || !img) return

    const r = cropRect.value
    const w = Math.max(1, Math.round(r.w))
    const h = Math.max(1, Math.round(r.h))

    const maxDim = 200
    const scale = Math.min(maxDim / w, maxDim / h, 1)
    canvas.width = Math.round(w * scale)
    canvas.height = Math.round(h * scale)

    const ctx = canvas.getContext('2d')!
    drawCheckerboard(ctx, canvas.width, canvas.height)
    ctx.drawImage(img, r.x, r.y, r.w, r.h, 0, 0, canvas.width, canvas.height)
  }

  function drawCheckerboard(ctx: CanvasRenderingContext2D, w: number, h: number) {
    const size = 8
    for (let y = 0; y < h; y += size) {
      for (let x = 0; x < w; x += size) {
        ctx.fillStyle = (Math.floor(x / size) + Math.floor(y / size)) % 2 === 0 ? '#2a2a3e' : '#1e1e30'
        ctx.fillRect(x, y, size, size)
      }
    }
  }

  // --- Crop Drag ---

  function getCanvasPos(e: PointerEvent) {
    const canvas = cropCanvas.value!
    const rect = canvas.getBoundingClientRect()
    return {
      x: (e.clientX - rect.left) / canvasScale,
      y: (e.clientY - rect.top) / canvasScale,
    }
  }

  function startCropDrag(e: PointerEvent) {
    const pos = getCanvasPos(e)
    const r = cropRect.value
    const handleThreshold = 12 / canvasScale

    if (Math.abs(pos.x - r.x) < handleThreshold && Math.abs(pos.y - r.y) < handleThreshold) {
      cropDragType.value = 'nw'
    } else if (Math.abs(pos.x - (r.x + r.w)) < handleThreshold && Math.abs(pos.y - r.y) < handleThreshold) {
      cropDragType.value = 'ne'
    } else if (Math.abs(pos.x - r.x) < handleThreshold && Math.abs(pos.y - (r.y + r.h)) < handleThreshold) {
      cropDragType.value = 'sw'
    } else if (Math.abs(pos.x - (r.x + r.w)) < handleThreshold && Math.abs(pos.y - (r.y + r.h)) < handleThreshold) {
      cropDragType.value = 'se'
    } else if (pos.x >= r.x && pos.x <= r.x + r.w && pos.y >= r.y && pos.y <= r.y + r.h) {
      cropDragType.value = 'move'
    } else {
      return
    }

    cropDragging.value = true
    cropDragStart.value = { x: pos.x, y: pos.y, rx: r.x, ry: r.y, rw: r.w, rh: r.h }
    cropCanvas.value?.setPointerCapture(e.pointerId)
  }

  function onCropDrag(e: PointerEvent) {
    if (!cropDragging.value || !processedImage.value) return
    const pos = getCanvasPos(e)
    const s = cropDragStart.value
    const dx = pos.x - s.x
    const dy = pos.y - s.y
    const imgW = processedImage.value.naturalWidth
    const imgH = processedImage.value.naturalHeight

    const r = { ...cropRect.value }

    switch (cropDragType.value) {
      case 'move':
        r.x = Math.max(0, Math.min(imgW - s.rw, s.rx + dx))
        r.y = Math.max(0, Math.min(imgH - s.rh, s.ry + dy))
        break
      case 'nw':
        r.x = Math.max(0, Math.min(s.rx + s.rw - 20, s.rx + dx))
        r.y = Math.max(0, Math.min(s.ry + s.rh - 20, s.ry + dy))
        r.w = s.rw - (r.x - s.rx)
        r.h = s.rh - (r.y - s.ry)
        break
      case 'ne':
        r.w = Math.max(20, Math.min(imgW - s.rx, s.rw + dx))
        r.y = Math.max(0, Math.min(s.ry + s.rh - 20, s.ry + dy))
        r.h = s.rh - (r.y - s.ry)
        break
      case 'sw':
        r.x = Math.max(0, Math.min(s.rx + s.rw - 20, s.rx + dx))
        r.w = s.rw - (r.x - s.rx)
        r.h = Math.max(20, Math.min(imgH - s.ry, s.rh + dy))
        break
      case 'se':
        r.w = Math.max(20, Math.min(imgW - s.rx, s.rw + dx))
        r.h = Math.max(20, Math.min(imgH - s.ry, s.rh + dy))
        break
    }

    cropRect.value = r
    drawCropCanvas()
    drawResultCanvas()
  }

  function endCropDrag() {
    cropDragging.value = false
    cropDragType.value = null
  }

  // Redraw when padding changes
  watch(cropPadding, () => {
    if (step.value === 'crop' || step.value === 'done') {
      autoCrop()
    }
  })

  // --- Coin Search ---

  function searchCoins() {
    if (searchTimeout) clearTimeout(searchTimeout)
    selectedCoinId.value = null
    if (!coinSearch.value.trim()) {
      coinOptions.value = []
      return
    }
    searchTimeout = setTimeout(async () => {
      coinsLoading.value = true
      try {
        const res = await getCoins({ search: coinSearch.value.trim(), limit: 20 })
        coinOptions.value = res.data.coins || []
      } catch {
        coinOptions.value = []
      } finally {
        coinsLoading.value = false
      }
    }, 300)
  }

  // --- Save / Download ---

  function getResultBlob(): Promise<Blob> {
    return new Promise((resolve) => {
      const img = processedImage.value!
      const r = cropRect.value
      const canvas = document.createElement('canvas')
      canvas.width = Math.round(r.w)
      canvas.height = Math.round(r.h)
      const ctx = canvas.getContext('2d')!
      ctx.drawImage(img, r.x, r.y, r.w, r.h, 0, 0, canvas.width, canvas.height)
      canvas.toBlob((blob) => resolve(blob!), 'image/png')
    })
  }

  async function saveToExisting(): Promise<number | null> {
    if (!selectedCoinId.value) return null
    saving.value = true
    saveMsg.value = ''
    saveError.value = false
    try {
      const blob = await getResultBlob()
      const file = new File([blob], `${saveImageType.value}.png`, { type: 'image/png' })
      const isPrimary = saveImageType.value === 'obverse'
      await uploadImage(selectedCoinId.value, file, saveImageType.value, isPrimary)
      const coin = coinOptions.value.find(c => c.id === selectedCoinId.value)
      saveMsg.value = `Saved as ${saveImageType.value} to "${coin?.name || 'coin'}"!`
      return selectedCoinId.value
    } catch {
      saveMsg.value = 'Failed to save image'
      saveError.value = true
      return null
    } finally {
      saving.value = false
    }
  }

  async function saveToNewCoin(): Promise<number | null> {
    if (!newCoinName.value.trim()) return null
    saving.value = true
    saveMsg.value = ''
    saveError.value = false
    try {
      const res = await createCoin({ name: newCoinName.value.trim() })
      const coin = res.data
      const blob = await getResultBlob()
      const file = new File([blob], 'obverse.png', { type: 'image/png' })
      await uploadImage(coin.id, file, 'obverse', true)
      saveMsg.value = `Created "${coin.name}" with obverse image!`
      return coin.id
    } catch {
      saveMsg.value = 'Failed to create coin'
      saveError.value = true
      return null
    } finally {
      saving.value = false
    }
  }

  async function downloadResult() {
    const blob = await getResultBlob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `coin-${saveImageType.value}-processed.png`
    a.click()
    URL.revokeObjectURL(url)
  }

  function reset() {
    if (processedImage.value) URL.revokeObjectURL(processedImage.value.src)
    sourceImage.value = null
    processedBlob.value = null
    processedImage.value = null
    step.value = 'preview'
    inputError.value = ''
    saveMsg.value = ''
  }

  onUnmounted(() => {
    if (processedImage.value) URL.revokeObjectURL(processedImage.value.src)
  })

  return {
    // Input state
    sourceImage,
    urlLoading,
    inputError,

    // Processing state
    step,
    processedBlob,

    // Crop state
    cropPadding,
    cropRect,

    // Save state
    saveTab,
    saveImageType,
    saving,
    saveMsg,
    saveError,

    // Coin selection
    coinSearch,
    coinOptions,
    coinsLoading,
    selectedCoinId,

    // New coin
    newCoinName,

    // Input methods
    loadImageFromFile,
    handleUrlLoad,

    // Processing methods
    removeBackground,

    // Crop methods
    autoCrop,
    resetCrop,
    startCropDrag,
    onCropDrag,
    endCropDrag,

    // Save methods
    saveToExisting,
    saveToNewCoin,
    downloadResult,
    reset,
    searchCoins,
  }
}
