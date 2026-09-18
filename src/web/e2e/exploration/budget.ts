import type {
  ExplorationLimits,
  ExplorationUsage,
  ModelUsage,
  TerminationReason,
} from './contracts'

type Reservable = 'steps' | 'modelCalls' | 'browserActions' | 'issueAttempts'
type Clock = { now: () => number }

const precedence: ReadonlyArray<keyof ExplorationLimits> = [
  'wallTimeSeconds',
  'modelTokens',
  'modelCalls',
  'browserActions',
  'steps',
  'issueAttempts',
]

const reasons: Record<keyof ExplorationLimits, TerminationReason> = {
  steps: 'step_limit',
  wallTimeSeconds: 'wall_time_limit',
  modelCalls: 'model_call_limit',
  modelTokens: 'model_token_limit',
  browserActions: 'browser_action_limit',
  issueAttempts: 'issue_attempt_limit',
}

export class BudgetExceededError extends Error {
  constructor(
    message: string,
    readonly reason: TerminationReason,
    readonly reachedLimits: Array<keyof ExplorationLimits>,
  ) {
    super(message)
    this.name = 'BudgetExceededError'
  }
}

export class BudgetGuard {
  readonly limits: Readonly<ExplorationLimits>
  readonly signal: AbortSignal
  private readonly startedAt: number
  private readonly now: () => number
  private readonly abortController = new AbortController()
  private readonly deadlineTimer: ReturnType<typeof setTimeout>
  private usage: ExplorationUsage = {
    steps: 0,
    modelCalls: 0,
    modelInputTokens: 0,
    modelOutputTokens: 0,
    modelTokens: 0,
    browserActions: 0,
    issueAttempts: 0,
    elapsedMilliseconds: 0,
  }
  private providerUsageInvalid = false

  constructor(limits: ExplorationLimits, clock: Clock = { now: () => performance.now() }) {
    this.limits = Object.freeze({ ...limits })
    this.now = clock.now
    this.startedAt = this.now()
    this.signal = this.abortController.signal
    this.deadlineTimer = setTimeout(() => {
      if (!this.signal.aborted) this.abortController.abort(this.limitError('wallTimeSeconds'))
    }, limits.wallTimeSeconds * 1000)
    this.deadlineTimer.unref?.()
  }

  reserve(kind: Reservable, count = 1): void {
    this.checkDeadline()
    if (this.providerUsageInvalid && kind === 'modelCalls') {
      throw new BudgetExceededError('Provider usage is invalid; refusing another model call', 'provider_usage_invalid', [])
    }
    if (kind === 'modelCalls' && this.usage.modelTokens >= this.limits.modelTokens) {
      throw this.limitError('modelTokens')
    }
    if (!Number.isInteger(count) || count < 1) throw new Error('Reservation count must be a positive integer')
    if (this.usage[kind] + count > this.limits[kind]) {
      throw this.limitError(kind)
    }
    this.usage[kind] += count
  }

  async perform<T>(kind: Reservable, operation: (signal: AbortSignal) => Promise<T>): Promise<T> {
    this.reserve(kind)
    return await this.withinDeadline(operation)
  }

  async withinDeadline<T>(operation: (signal: AbortSignal) => Promise<T>): Promise<T> {
    this.checkDeadline()
    if (this.signal.aborted) throw this.signal.reason
    const aborted = new Promise<never>((_resolve, reject) => {
      this.signal.addEventListener('abort', () => reject(this.signal.reason), { once: true })
    })
    try {
      return await Promise.race([operation(this.signal), aborted])
    } finally {
      this.checkDeadline()
    }
  }

  reconcileModelUsage(usage: ModelUsage | undefined): void {
    if (
      usage === undefined ||
      !Number.isInteger(usage.inputTokens) ||
      !Number.isInteger(usage.outputTokens) ||
      usage.inputTokens < 0 ||
      usage.outputTokens < 0
    ) {
      this.providerUsageInvalid = true
      throw new BudgetExceededError('Provider usage is missing or invalid', 'provider_usage_invalid', [])
    }
    const total = usage.inputTokens + usage.outputTokens
    if (this.usage.modelTokens + total > this.limits.modelTokens) {
      throw this.limitError('modelTokens')
    }
    this.usage.modelInputTokens += usage.inputTokens
    this.usage.modelOutputTokens += usage.outputTokens
    this.usage.modelTokens += total
    this.checkDeadline()
  }

  checkDeadline(): void {
    this.usage.elapsedMilliseconds = Math.max(0, Math.floor(this.now() - this.startedAt))
    if (this.usage.elapsedMilliseconds >= this.limits.wallTimeSeconds * 1000) {
      const error = this.limitError('wallTimeSeconds')
      if (!this.signal.aborted) this.abortController.abort(error)
      throw error
    }
  }

  throwIfReached(): void {
    this.checkDeadline()
    const reached = this.reachedLimits()
    if (reached.length > 0) throw this.limitError(reached[0])
  }

  dispose(): void {
    clearTimeout(this.deadlineTimer)
  }

  termination(): { reason: TerminationReason; reachedLimits: Array<keyof ExplorationLimits> } {
    const reached = this.reachedLimits()
    return {
      reason: reached.length === 0 ? 'completed' : reasons[reached[0]],
      reachedLimits: reached,
    }
  }

  snapshot(): { limits: Readonly<ExplorationLimits>; usage: Readonly<ExplorationUsage> } {
    this.usage.elapsedMilliseconds = Math.max(0, Math.floor(this.now() - this.startedAt))
    return Object.freeze({
      limits: this.limits,
      usage: Object.freeze({ ...this.usage }),
    })
  }

  private reachedLimits(): Array<keyof ExplorationLimits> {
    return precedence.filter((key) => {
      if (key === 'wallTimeSeconds') return this.usage.elapsedMilliseconds >= this.limits.wallTimeSeconds * 1000
      if (key === 'modelTokens') return this.usage.modelTokens >= this.limits.modelTokens
      return this.usage[key] >= this.limits[key]
    })
  }

  private limitError(kind: keyof ExplorationLimits): BudgetExceededError {
    const reached = this.reachedLimits()
    if (!reached.includes(kind)) reached.push(kind)
    reached.sort((a, b) => precedence.indexOf(a) - precedence.indexOf(b))
    return new BudgetExceededError(`${kind} limit reached`, reasons[reached[0]], reached)
  }
}
