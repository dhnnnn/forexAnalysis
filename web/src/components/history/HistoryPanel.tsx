import { useHistoryStore } from '../../stores/historyStore'
import { useResizable } from '../../hooks/useResizable'
import { SignalsTab } from './SignalsTab'
import { PerformanceTab } from './PerformanceTab'
import { RulesTab } from './RulesTab'
import { RegimeTab } from './RegimeTab'
import { SystemLogTab } from './SystemLogTab'
import type { HistoryTab } from '../../types/history'
import { ChevronDown, ChevronUp } from 'lucide-react'

const TABS: { id: HistoryTab; label: string }[] = [
  { id: 'signals',     label: 'Signals' },
  { id: 'performance', label: 'Performance' },
  { id: 'rules',       label: 'Rules' },
  { id: 'regime',      label: 'Regime' },
  { id: 'system',      label: 'System Log' },
]

export function HistoryPanel() {
  const {
    activeTab, setActiveTab,
    panelHeight, setPanelHeight,
    isCollapsed, toggleCollapse,
  } = useHistoryStore()

  const { onMouseDown } = useResizable({
    initialHeight: panelHeight,
    minHeight: 150,
    maxHeight: Math.round(window.innerHeight * 0.5),
    onHeightChange: setPanelHeight,
  })

  const effectiveHeight = isCollapsed ? 36 : panelHeight

  return (
    <div
      className="flex flex-col border-t border-border-subtle bg-bg-secondary flex-shrink-0 transition-[height] duration-200"
      style={{ height: effectiveHeight }}
      aria-label="History Panel"
    >
      {/* Resize handle */}
      <div
        className="resize-handle"
        onMouseDown={onMouseDown}
        onDoubleClick={toggleCollapse}
        title="Drag to resize | Double-click to collapse"
      />

      {/* Tab bar */}
      <div className="flex items-center gap-1 px-3 border-b border-border-subtle flex-shrink-0 h-9">
        {!isCollapsed && TABS.map((tab) => (
          <button
            key={tab.id}
            id={`history-tab-${tab.id}`}
            onClick={() => setActiveTab(tab.id)}
            className={activeTab === tab.id ? 'tab-btn-active' : 'tab-btn'}
          >
            {tab.label}
          </button>
        ))}

        {/* Collapse toggle */}
        <button
          id="history-collapse-btn"
          onClick={toggleCollapse}
          className="ml-auto text-text-muted hover:text-text-primary transition-colors p-1"
          aria-label={isCollapsed ? 'Expand history panel' : 'Collapse history panel'}
        >
          {isCollapsed ? <ChevronUp size={12} /> : <ChevronDown size={12} />}
        </button>
      </div>

      {/* Tab content */}
      {!isCollapsed && (
        <div className="flex-1 min-h-0 overflow-hidden">
          {activeTab === 'signals'     && <SignalsTab />}
          {activeTab === 'performance' && <PerformanceTab />}
          {activeTab === 'rules'       && <RulesTab />}
          {activeTab === 'regime'      && <RegimeTab />}
          {activeTab === 'system'      && <SystemLogTab />}
        </div>
      )}
    </div>
  )
}
