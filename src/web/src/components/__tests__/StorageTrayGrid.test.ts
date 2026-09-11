import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import StorageTrayGrid from '@/components/tray/StorageTrayGrid.vue'
import { occupiedThreeByThreeTray, twentyByTwentyTray } from '@/test/fixtures/storageTrays'

describe('StorageTrayGrid', () => {
  it('materializes exact row-major geometry including empty wells', () => {
    const wrapper = mount(StorageTrayGrid, { props: { tray: occupiedThreeByThreeTray } })
    expect(wrapper.find('[role="grid"]').attributes('aria-label')).toBe(
      'Cabinet A, 3 rows by 3 columns, 1 of 9 occupied'
    )
    const cells = wrapper.findAll('[role="gridcell"]')
    expect(cells).toHaveLength(9)
    expect(cells[5]?.text()).toContain('2,3')
    expect(cells[5]?.find('[role="button"]').attributes('aria-label')).toBe('Denarius, row 2, column 3')
    expect(cells[0]?.attributes('aria-label')).toBe('Empty, row 1, column 1')
  })

  it('keeps all 400 wells and the persisted 20-column geometry', () => {
    const wrapper = mount(StorageTrayGrid, { props: { tray: twentyByTwentyTray } })
    expect(wrapper.findAll('[role="gridcell"]')).toHaveLength(400)
    expect(wrapper.find('[role="grid"]').attributes('style')).toContain('repeat(20, 44px)')
  })

  it('activates occupied wells with Space', async () => {
    const wrapper = mount(StorageTrayGrid, { props: { tray: occupiedThreeByThreeTray } })
    await wrapper.find('[role="button"]').trigger('keydown.space')
    expect(wrapper.emitted('coin-clicked')).toEqual([[42]])
  })
})
