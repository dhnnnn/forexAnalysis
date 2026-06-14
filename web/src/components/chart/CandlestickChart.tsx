import { useEffect, useRef } from 'react'
import {
  createChart,
  type IChartApi,
  type ISeriesApi,
  ColorType,
  CrosshairMode,
} from 'lightweight-charts'
import { useQuery, useSubscription } from '@apollo/client'
import { GET_CANDLES } from '../../graphql/queries'
import { CANDLE_UPDATED } from '../../graphql/subscriptions'
import { useChartStore } from '../../stores/chartStore'
import { useConnectionStore } from '../../stores/connectionStore'
import { toUnixTimestamp } from '../../utils/formatters'
import { CANDLE_UP_COLOR, CANDLE_DOWN_COLOR } from '../../utils/colors'
import type { CandleData } from '../../types/candle'
import { DEFAULT_CANDLE_LIMIT } from '../../utils/constants'
import { IndicatorBar } from './IndicatorBar'

const CHART_THEME = {
  background:   { type: ColorType.Solid, color: '#0d1117' },
  textColor:    '#8b949e',
  gridLines:    '#1c2128',
  borderColor:  '#30363d',
  crosshairColor: '#484f58',
}

export function CandlestickChart() {
  const containerRef = useRef<HTMLDivElement>(null)
  const chartRef     = useRef<IChartApi | null>(null)
  const candleRef    = useRef<ISeriesApi<'Candlestick'> | null>(null)
  const volumeRef    = useRef<ISeriesApi<'Histogram'> | null>(null)

  const activePair   = useConnectionStore((s) => s.activePair)
  const timeframe    = useConnectionStore((s) => s.timeframe)
  const { addCandle, setCandles, candles } = useChartStore()

  // Fetch historical candles
  const { data: historicalData } = useQuery(GET_CANDLES, {
    variables: { pair: activePair, timeframe, limit: DEFAULT_CANDLE_LIMIT },
    fetchPolicy: 'network-only',
  })

  // Subscribe to real-time candle updates
  useSubscription(CANDLE_UPDATED, {
    variables: { pair: activePair },
    onData: ({ data }) => {
      const candle = data.data?.candleUpdated as CandleData | undefined
      if (!candle) return
      addCandle(candle)
      // Update chart directly
      if (candleRef.current) {
        const t = toUnixTimestamp(candle.timestamp) as unknown as import('lightweight-charts').UTCTimestamp
        candleRef.current.update({ time: t, open: candle.open, high: candle.high, low: candle.low, close: candle.close })
      }
      if (volumeRef.current) {
        const t = toUnixTimestamp(candle.timestamp) as unknown as import('lightweight-charts').UTCTimestamp
        volumeRef.current.update({
          time: t,
          value: candle.volume,
          color: candle.close >= candle.open ? `${CANDLE_UP_COLOR}80` : `${CANDLE_DOWN_COLOR}80`,
        })
      }
    },
  })

  // Initialize chart
  useEffect(() => {
    if (!containerRef.current) return

    const chart = createChart(containerRef.current, {
      layout: {
        background: CHART_THEME.background,
        textColor:  CHART_THEME.textColor,
        fontFamily: "'JetBrains Mono', monospace",
      },
      grid: {
        vertLines: { color: CHART_THEME.gridLines },
        horzLines: { color: CHART_THEME.gridLines },
      },
      crosshair: {
        mode: CrosshairMode.Normal,
        vertLine: { color: CHART_THEME.crosshairColor, width: 1, style: 3 },
        horzLine: { color: CHART_THEME.crosshairColor, width: 1, style: 3 },
      },
      rightPriceScale: {
        borderColor: CHART_THEME.borderColor,
        scaleMargins: { top: 0.05, bottom: 0.25 },
      },
      timeScale: {
        borderColor: CHART_THEME.borderColor,
        timeVisible: true,
        secondsVisible: false,
        fixLeftEdge: false,
        fixRightEdge: false,
      },
      width:  containerRef.current.clientWidth,
      height: containerRef.current.clientHeight,
    })

    const candleSeries = chart.addCandlestickSeries({
      upColor:         CANDLE_UP_COLOR,
      downColor:       CANDLE_DOWN_COLOR,
      borderUpColor:   CANDLE_UP_COLOR,
      borderDownColor: CANDLE_DOWN_COLOR,
      wickUpColor:     CANDLE_UP_COLOR,
      wickDownColor:   CANDLE_DOWN_COLOR,
    })

    const volumeSeries = chart.addHistogramSeries({
      priceFormat: { type: 'volume' },
      priceScaleId: 'volume',
    })
    chart.priceScale('volume').applyOptions({
      scaleMargins: { top: 0.82, bottom: 0 },
    })

    chartRef.current  = chart
    candleRef.current = candleSeries
    volumeRef.current = volumeSeries

    // Resize observer
    const ro = new ResizeObserver(() => {
      if (containerRef.current) {
        chart.applyOptions({
          width:  containerRef.current.clientWidth,
          height: containerRef.current.clientHeight,
        })
      }
    })
    ro.observe(containerRef.current)

    return () => {
      ro.disconnect()
      chart.remove()
      chartRef.current  = null
      candleRef.current = null
      volumeRef.current = null
    }
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  // Load historical data into chart
  useEffect(() => {
    const raw = historicalData?.candles as CandleData[] | undefined
    if (!raw || !candleRef.current || !volumeRef.current) return

    setCandles(activePair, raw)

    const candleData = raw.map((c) => ({
      time:  toUnixTimestamp(c.timestamp) as unknown as import('lightweight-charts').UTCTimestamp,
      open:  c.open,
      high:  c.high,
      low:   c.low,
      close: c.close,
    })).sort((a, b) => (a.time as number) - (b.time as number))

    const volumeData = raw.map((c) => ({
      time:  toUnixTimestamp(c.timestamp) as unknown as import('lightweight-charts').UTCTimestamp,
      value: c.volume,
      color: c.close >= c.open ? `${CANDLE_UP_COLOR}70` : `${CANDLE_DOWN_COLOR}70`,
    })).sort((a, b) => (a.time as number) - (b.time as number))

    candleRef.current.setData(candleData)
    volumeRef.current.setData(volumeData)
    chartRef.current?.timeScale().fitContent()
  }, [historicalData, activePair]) // eslint-disable-line react-hooks/exhaustive-deps

  // Last candle for indicator bar
  const pairCandles = candles[activePair] ?? []
  const lastCandle  = pairCandles[pairCandles.length - 1]

  return (
    <div className="flex flex-col h-full">
      {/* Chart */}
      <div ref={containerRef} className="chart-container flex-1 min-h-0" />

      {/* Indicator bar */}
      <IndicatorBar pair={activePair} lastCandle={lastCandle} />
    </div>
  )
}
