// ── Knowledge Types ───────────────────────────────────────────────────────
export interface KnowledgeRule {
  id: string
  sourceAgent: string
  targetAgent: string
  regime: string
  weightDelta: number
  minWeight: number
  confidence: number
  reasoning: string
  applyCount: number
  createdAt: string
  expiresAt: string
  status: 'active' | 'expired'
  impactAccuracyDelta?: number | null
}

export interface PipelineEvent {
  type: 'START' | 'END'
  pair: string
  timestamp: string
  durationMs?: number | null
}
