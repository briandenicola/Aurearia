// Friendly Coin Copilot helper names. Must match _TOOL_LABELS in
// src/agent/app/teams/coin_copilot.py so the checklist and result lines agree;
// copilotToolLabels.test.ts fails if the two drift.
export const COPILOT_TOOL_LABELS: Readonly<Record<string, string>> = {
  search_my_collection: 'Search owned collection',
  get_coin: 'Read coin details',
  collection_summary: 'Summarize collection',
  top_coins_by_value: 'Review top recorded values',
  portfolio_review: 'Review collection portfolio',
  gap_analysis: 'Analyze collection gaps',
  market_search: 'Search dealer listings',
  auction_search: 'Search auction lots',
  price_trends: 'Analyze completed-sale price trends',
  similar_lots: 'Find similar auction lots',
  deep_analysis_handoff: 'Use existing Deep Analysis',
}

// Unknown helpers get a generic name rather than an internal tool name.
export function copilotToolLabel(name: string): string {
  return COPILOT_TOOL_LABELS[name] ?? 'Copilot helper'
}
