const MAX_GALLERY_IMAGE_DIMENSION = 1920
const GALLERY_JPEG_QUALITY = 0.85

function loadImage(source: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const image = new Image()
    image.onload = () => resolve(image)
    image.onerror = () => reject(new Error('The selected image could not be opened.'))
    image.src = source
  })
}

function canvasToJPEG(canvas: HTMLCanvasElement): Promise<Blob> {
  return new Promise((resolve, reject) => {
    canvas.toBlob((blob) => {
      if (blob) {
        resolve(blob)
        return
      }
      reject(new Error('The selected image could not be converted to JPEG.'))
    }, 'image/jpeg', GALLERY_JPEG_QUALITY)
  })
}

function jpegFilename(filename: string): string {
  const stem = filename.replace(/\.[^.]+$/, '') || 'coin'
  return `${stem}.jpg`
}

export async function normalizeGalleryImage(file: File): Promise<File> {
  const source = URL.createObjectURL(file)
  try {
    const image = await loadImage(source)
    const width = image.naturalWidth
    const height = image.naturalHeight
    if (width <= 0 || height <= 0) {
      throw new Error('The selected image has invalid dimensions.')
    }

    const scale = Math.min(1, MAX_GALLERY_IMAGE_DIMENSION / Math.max(width, height))
    const canvas = document.createElement('canvas')
    canvas.width = Math.max(1, Math.round(width * scale))
    canvas.height = Math.max(1, Math.round(height * scale))

    const context = canvas.getContext('2d')
    if (!context) {
      throw new Error('This browser could not prepare the selected image.')
    }
    context.drawImage(image, 0, 0, canvas.width, canvas.height)

    const blob = await canvasToJPEG(canvas)
    return new File([blob], jpegFilename(file.name), {
      type: 'image/jpeg',
      lastModified: file.lastModified,
    })
  } finally {
    URL.revokeObjectURL(source)
  }
}

