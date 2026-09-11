import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import StorageTrayGrid from '@/components/tray/StorageTrayGrid.vue'
import MuseumTrayWell from '@/components/tray/MuseumTrayWell.vue'
import { occupiedThreeByThreeTray, twentyByTwentyTray } from '@/test/fixtures/storageTrays'

describe('StorageTrayGrid', () => {
  it('materializes exact row-major geometry including empty wells', () => {
    const wrapper = mount(StorageTrayGrid, { props: { tray: occupiedThreeByThreeTray, feltTheme: 'navy' } })
    expect(wrapper.find('[role="grid"]').attributes('aria-label')).toBe(
      'Cabinet A, 3 rows by 3 columns, 1 of 9 occupied'
    )
    expect(wrapper.find('.tray-surface').classes()).toContain('felt-navy')
    const cells = wrapper.findAll('[role="gridcell"]')
    expect(cells).toHaveLength(9)
    expect(cells[5]?.text()).toContain('2,3')
    expect(cells[5]?.find('[role="button"]').attributes('style')).toContain('width: 76px')
    expect(cells[5]?.find('[role="button"]').attributes('aria-label')).toBe('Denarius, row 2, column 3')
    expect(cells[5]?.find('.tray-name').text()).toBe('Denarius')
    expect(cells[0]?.attributes('aria-label')).toBe('Empty, row 1, column 1')
    expect(cells[0]?.find('.tray-name').exists()).toBe(true)
    expect(cells[0]?.find('.tray-name').text()).toBe('')
  })

  it('keeps all 400 wells and the persisted 20-column geometry', () => {
    const wrapper = mount(StorageTrayGrid, { props: { tray: twentyByTwentyTray, feltTheme: 'red' } })
    expect(wrapper.findAll('[role="gridcell"]')).toHaveLength(400)
    expect(wrapper.find('[role="grid"]').attributes('style')).toContain('repeat(20, var(--storage-well-size))')
  })

  it('activates occupied wells with Space', async () => {
    const wrapper = mount(StorageTrayGrid, { props: { tray: occupiedThreeByThreeTray, feltTheme: 'green' } })
    await wrapper.find('[role="button"]').trigger('keydown.space')
    expect(wrapper.emitted('coin-clicked')).toEqual([[42]])
  })

  it('toggles every occupied well from obverse to reverse', async () => {
    const wrapper = mount(StorageTrayGrid, { props: { tray: occupiedThreeByThreeTray, feltTheme: 'green' } })
    const occupiedWell = wrapper.findAllComponents(MuseumTrayWell).find(well => well.props('interactive'))

    expect(occupiedWell?.props('preferredFace')).toBe('obverse')
    expect(occupiedWell?.props('wellStageSizePx')).toBe(76)

    await wrapper.find('.tray-face-toggle').trigger('click')

    expect(occupiedWell?.props('preferredFace')).toBe('reverse')
    expect(wrapper.find('.tray-face-toggle').attributes('aria-pressed')).toBe('true')
  })
})
