export const SET_COLOR_PALETTE = [
  '#c9a84c',
  '#b08d57',
  '#9b59b6',
  '#6b8e23',
  '#c0392b',
  '#4682b4',
  '#10b981',
  '#8b5cf6',
] as const

export function randomSetColor(): string {
  return SET_COLOR_PALETTE[Math.floor(Math.random() * SET_COLOR_PALETTE.length)] ?? SET_COLOR_PALETTE[0]
}
