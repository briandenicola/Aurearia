import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import BulkLocationPickerModal from '../BulkLocationPickerModal.vue'
import type { StorageLocation } from '@/types'

const locations: StorageLocation[] = [
  { id: 1, name: 'Cabinet', type: 'standard', rows: null, columns: null, occupied: 0, capacity: 0 },
  { id: 2, name: 'Tray A', type: 'tray', rows: 3, columns: 3, occupied: 0, capacity: 9 },
]

describe('BulkLocationPickerModal', () => {
  it('allows clear and Standard choices but disables trays with the per-coin explanation', async () => {
    const wrapper = mount(BulkLocationPickerModal, {
      props: { open: true, locations },
      global: { stubs: { Teleport: true, MapPin: true } },
    })
    const buttons = wrapper.findAll('button')
    expect(wrapper.text()).toContain('No location')
    expect(wrapper.text()).toContain('Tray assignments require choosing a slot on each coin.')
    expect(buttons.find((button) => button.text().includes('Tray A'))?.attributes('disabled')).toBeDefined()

    await buttons.find((button) => button.text().includes('Cabinet'))!.trigger('click')
    await buttons.find((button) => button.text().includes('No location'))!.trigger('click')
    expect(wrapper.emitted('select')).toEqual([[1], [null]])
  })
})
