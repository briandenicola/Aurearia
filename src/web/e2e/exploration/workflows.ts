import { F013_WORKFLOW_INVENTORY, type F013Workflow } from '../fixtures/workflow'
import type { WorkflowId } from './contracts'

export function selectExplorationWorkflows(ids: readonly WorkflowId[]): readonly F013Workflow[] {
  const unique = new Set(ids)
  if (unique.size !== ids.length) throw new Error('Workflow selection contains duplicates')
  return Object.freeze(ids.map((id) => {
    const workflow = F013_WORKFLOW_INVENTORY[id]
    if (!workflow) throw new Error(`Unknown F013 workflow: ${id}`)
    return workflow
  }))
}
