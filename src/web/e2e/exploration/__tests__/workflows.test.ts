import { describe, expect, it } from 'vitest'
import { F013_WORKFLOW_INVENTORY } from '../../fixtures/workflow'
import { selectExplorationWorkflows } from '../workflows'
import { WORKFLOW_IDS } from '../contracts'

describe('F013 workflow adapter', () => {
  it('maps every baseline ID to the canonical immutable inventory', () => {
    expect(Object.keys(F013_WORKFLOW_INVENTORY)).toEqual([...WORKFLOW_IDS])
    expect(Object.isFrozen(F013_WORKFLOW_INVENTORY)).toBe(true)
    for (const workflow of Object.values(F013_WORKFLOW_INVENTORY)) {
      expect(workflow.routes.length).toBeGreaterThan(0)
      expect(workflow.goal.length).toBeGreaterThan(0)
      expect(workflow.checkpoints.length).toBeGreaterThan(0)
    }
  })

  it('uses the canonical application edit route', () => {
    for (const id of ['edit-one-field', 'storage-location-change-clear', 'image-upload-delete', 'mobile-edit'] as const) {
      expect(F013_WORKFLOW_INVENTORY[id].routes).toContain('/edit/1')
      expect(F013_WORKFLOW_INVENTORY[id].routes).not.toContain('/coin/1/edit')
    }
  })

  it('selects only requested workflows without copying fixture data', () => {
    const selected = selectExplorationWorkflows(['login-session', 'mobile-edit'])
    expect(selected.map((item) => item.id)).toEqual(['login-session', 'mobile-edit'])
    expect(selected[0]).toBe(F013_WORKFLOW_INVENTORY['login-session'])
    expect(selected[1].viewport).toEqual({ width: 390, height: 844 })
  })
})
