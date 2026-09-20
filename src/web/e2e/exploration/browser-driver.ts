import type { Locator, Page, Route } from '@playwright/test'
import type { ActionName } from './contracts'

const ACTIONS: readonly ActionName[] = [
  'navigate', 'click', 'fill', 'select', 'upload_fixture',
  'set_viewport', 'back', 'wait_for_ui', 'checkpoint', 'finish',
]
const INJECTION = /(ignore (all|previous) (rules|instructions)|shell command|\/etc\/|powershell|cmd\.exe|document\.cookie)/i

export interface BrowserAction {
  action: ActionName
  target: {
    route?: string
    role?: string
    accessibleName?: string
    label?: string
    value?: string
    fixtureId?: string
    width?: number
    height?: number
    milliseconds?: number
  } | null
}

export interface BrowserDriverContext {
  origin: string
  allowedRoutes: readonly string[]
  uploadFixtures: Readonly<Record<string, Buffer>>
}

export function validateNavigationDestination(urlValue: string, context: BrowserDriverContext): void {
  const destination = new URL(urlValue, context.origin)
  const origin = new URL(context.origin)
  if (destination.origin !== origin.origin || !context.allowedRoutes.includes(destination.pathname)) {
    throw new Error('Navigation escaped the selected workflow')
  }
}

export function validateBrowserAction(value: unknown, context: BrowserDriverContext): BrowserAction {
  if (typeof value !== 'object' || value === null) throw new Error('Action must be an object')
  const action = value as { action?: unknown; target?: unknown }
  if (typeof action.action !== 'string' || !ACTIONS.includes(action.action as ActionName)) {
    throw new Error('Action is not in the fixed vocabulary')
  }
  const target = action.target
  if (['back', 'checkpoint', 'finish'].includes(action.action)) {
    if (target != null) throw new Error(`${action.action} requires a null target`)
    return action as BrowserAction
  }
  if (typeof target !== 'object' || target === null || Array.isArray(target)) throw new Error('Action target is required')
  const data = target as Record<string, unknown>
  const allowedTargetKeys = ['route', 'role', 'accessibleName', 'label', 'value', 'fixtureId', 'width', 'height', 'milliseconds']
  if (Object.keys(data).some((key) => !allowedTargetKeys.includes(key))) throw new Error('Unknown target capability')
  const texts = Object.values(data).filter((item): item is string => typeof item === 'string')
  if (texts.some((item) => INJECTION.test(item))) throw new Error('Prompt-injected capability request refused')
  if (typeof data.label === 'string' && data.label.length > 200) throw new Error('Locator exceeds maximum')
  if (typeof data.accessibleName === 'string' && data.accessibleName.length > 200) throw new Error('Locator exceeds maximum')
  if (typeof data.role === 'string' && data.role.length > 80) throw new Error('Role exceeds maximum')
  if (typeof data.value === 'string' && data.value.length > 1000) throw new Error('Value exceeds maximum')

  if (action.action === 'navigate') {
    if (typeof data.route !== 'string' || !data.route.startsWith('/') || data.route.startsWith('//')) {
      throw new Error('Navigation must use a same-origin relative route')
    }
    if (!context.allowedRoutes.includes(data.route)) throw new Error('Route is outside selected workflow')
  }
  if (['click', 'fill', 'select'].includes(action.action) && !data.label && !(data.role && data.accessibleName)) {
    throw new Error('A bounded accessible locator is required')
  }
  if (['fill', 'select'].includes(action.action) && typeof data.value !== 'string') throw new Error('Value is required')
  if (action.action === 'upload_fixture') {
    if (typeof data.fixtureId !== 'string' || !(data.fixtureId in context.uploadFixtures)) {
      throw new Error('Only registered synthetic fixtures may be uploaded')
    }
  }
  if (action.action === 'set_viewport') {
    if (!Number.isInteger(data.width) || Number(data.width) < 320 || Number(data.width) > 1920 ||
        !Number.isInteger(data.height) || Number(data.height) < 568 || Number(data.height) > 1080) {
      throw new Error('Viewport is outside bounds')
    }
  }
  if (action.action === 'wait_for_ui' &&
      (!Number.isInteger(data.milliseconds) || Number(data.milliseconds) < 0 || Number(data.milliseconds) > 5000)) {
    throw new Error('Wait is outside bounds')
  }
  return action as BrowserAction
}

function locator(page: Page, target: NonNullable<BrowserAction['target']>): Locator {
  if (target.label) return page.getByLabel(target.label)
  return page.getByRole(target.role as never, { name: target.accessibleName })
}

async function withNavigationGuard<T>(
  page: Page,
  context: BrowserDriverContext,
  operation: () => Promise<T>,
): Promise<T> {
  let blocked: Error | undefined
  const handler = async (route: Route) => {
    const request = route.request()
    if (request.isNavigationRequest() && request.frame() === page.mainFrame()) {
      try {
        validateNavigationDestination(request.url(), context)
      } catch (error) {
        blocked = error instanceof Error ? error : new Error('Navigation refused')
        await route.abort('blockedbyclient')
        return
      }
    }
    await route.continue()
  }
  await page.route('**/*', handler)
  try {
    const result = await operation()
    if (blocked) throw blocked
    validateNavigationDestination(page.url(), context)
    return result
  } finally {
    await page.unroute('**/*', handler)
  }
}

export async function executeBrowserAction(
  page: Page,
  value: unknown,
  context: BrowserDriverContext,
): Promise<'completed' | 'checkpoint' | 'finished'> {
  const command = validateBrowserAction(value, context)
  const target = command.target
  switch (command.action) {
    case 'navigate':
      await withNavigationGuard(page, context, async () => {
        await page.goto(new URL(target!.route!, context.origin).toString())
      })
      return 'completed'
    case 'click':
      await withNavigationGuard(page, context, async () => locator(page, target!).click())
      return 'completed'
    case 'fill':
      await withNavigationGuard(page, context, async () => locator(page, target!).fill(target!.value!))
      return 'completed'
    case 'select':
      await withNavigationGuard(page, context, async () => locator(page, target!).selectOption(target!.value!))
      return 'completed'
    case 'upload_fixture':
      await withNavigationGuard(page, context, async () => {
        await locator(page, target!).setInputFiles({
          name: `${target!.fixtureId}.png`,
          mimeType: 'image/png',
          buffer: context.uploadFixtures[target!.fixtureId!],
        })
      })
      return 'completed'
    case 'set_viewport':
      await page.setViewportSize({ width: target!.width!, height: target!.height! })
      return 'completed'
    case 'back':
      await withNavigationGuard(page, context, async () => page.goBack())
      return 'completed'
    case 'wait_for_ui':
      await page.waitForTimeout(target!.milliseconds!)
      return 'completed'
    case 'checkpoint':
      return 'checkpoint'
    case 'finish':
      return 'finished'
  }
}
