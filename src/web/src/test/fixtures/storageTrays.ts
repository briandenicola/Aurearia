import type { TrayAggregate } from '@/types'

export function buildStorageTray(overrides: Partial<TrayAggregate> = {}): TrayAggregate {
  return {
    id: 1,
    name: 'Cabinet A',
    rows: 3,
    columns: 3,
    capacity: 9,
    occupied: 0,
    coins: [],
    ...overrides,
  }
}

export const oneByOneTray = buildStorageTray({ rows: 1, columns: 1, capacity: 1 })
export const emptyThreeByThreeTray = buildStorageTray()
export const occupiedThreeByThreeTray = buildStorageTray({
  occupied: 1,
  coins: [{
    id: 42,
    name: 'Denarius',
    diameterMm: 19,
    storageSlot: 6,
    images: [
      { filePath: 'denarius.jpg', imageType: 'obverse' },
      { filePath: 'denarius-reverse.jpg', imageType: 'reverse' },
    ],
  }],
})
export const twentyByTwentyTray = buildStorageTray({ id: 20, name: 'Large Tray', rows: 20, columns: 20, capacity: 400 })
export const missingImageTray = buildStorageTray({
  occupied: 1,
  coins: [{ id: 43, name: 'Image Missing', diameterMm: null, storageSlot: 1, images: [] }],
})
