'use client'

import { useEffect, useRef } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { 
  ArrowRight, 
  ArrowLeft, 
  Check, 
  X, 
  Wifi, 
  Server,
  Zap,
  Radio,
  AlertCircle
} from 'lucide-react'

interface NetworkEvent {
  id: string
  type: 'STUN' | 'OFFER' | 'ANSWER' | 'PUNCH' | 'CONNECTED' | 'FAILED' | 'PEER_JOINED' | 'PEER_LEFT' | 'INFO'
  message: string
  timestamp: number
  details?: string
}

interface NetworkMonitorProps {
  events: NetworkEvent[]
  maxEvents?: number
  className?: string
}

export function NetworkMonitor({ 
  events, 
  maxEvents = 50,
  className = '' 
}: NetworkMonitorProps) {
  const scrollRef = useRef<HTMLDivElement>(null)

  // Auto-scroll to bottom on new events
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [events])

  const getEventIcon = (type: NetworkEvent['type']) => {
    switch (type) {
      case 'STUN': return <Server className="w-3.5 h-3.5" />
      case 'OFFER': return <ArrowRight className="w-3.5 h-3.5" />
      case 'ANSWER': return <ArrowLeft className="w-3.5 h-3.5" />
      case 'PUNCH': return <Zap className="w-3.5 h-3.5" />
      case 'CONNECTED': return <Check className="w-3.5 h-3.5" />
      case 'FAILED': return <X className="w-3.5 h-3.5" />
      case 'PEER_JOINED': return <Wifi className="w-3.5 h-3.5" />
      case 'PEER_LEFT': return <Radio className="w-3.5 h-3.5" />
      default: return <AlertCircle className="w-3.5 h-3.5" />
    }
  }

  const getEventColor = (type: NetworkEvent['type']) => {
    switch (type) {
      case 'STUN': return 'text-signal-purple'
      case 'OFFER': return 'text-signal-cyan'
      case 'ANSWER': return 'text-signal-cyan'
      case 'PUNCH': return 'text-signal-orange'
      case 'CONNECTED': return 'text-signal-green'
      case 'FAILED': return 'text-signal-red'
      case 'PEER_JOINED': return 'text-signal-green'
      case 'PEER_LEFT': return 'text-altair-text-dim'
      default: return 'text-altair-text-dim'
    }
  }

  const formatTime = (timestamp: number) => {
    const date = new Date(timestamp)
    return date.toLocaleTimeString('en-US', { 
      hour12: false, 
      hour: '2-digit', 
      minute: '2-digit', 
      second: '2-digit',
    })
  }

  const displayEvents = events.slice(-maxEvents)

  return (
    <div className={`terminal-card flex flex-col ${className}`}>
      {/* Header */}
      <div className="terminal-card-header">
        <div className="terminal-dot bg-conn-failed" />
        <div className="terminal-dot bg-conn-pending" />
        <div className="terminal-dot bg-conn-active" />
        <span className="ml-2 text-xs text-altair-text-dim font-mono">network-monitor</span>
      </div>

      {/* Event log */}
      <div 
        ref={scrollRef}
        className="flex-1 overflow-y-auto p-4 font-mono text-xs space-y-1 min-h-[200px] max-h-[400px]"
      >
        <AnimatePresence initial={false}>
          {displayEvents.length === 0 ? (
            <div className="text-altair-muted text-center py-8">
              Waiting for network events...
            </div>
          ) : (
            displayEvents.map((event) => (
              <motion.div
                key={event.id}
                initial={{ opacity: 0, x: -10, height: 0 }}
                animate={{ opacity: 1, x: 0, height: 'auto' }}
                exit={{ opacity: 0, height: 0 }}
                transition={{ duration: 0.15 }}
                className="flex items-start gap-2 py-0.5"
              >
                {/* Timestamp */}
                <span className="text-altair-muted shrink-0 w-[72px]">
                  [{formatTime(event.timestamp)}]
                </span>

                {/* Type badge */}
                <span className={`shrink-0 flex items-center gap-1 w-[90px] ${getEventColor(event.type)}`}>
                  {getEventIcon(event.type)}
                  <span className="uppercase text-[10px] font-semibold">{event.type}</span>
                </span>

                {/* Message */}
                <span className="text-altair-text flex-1">
                  {event.message}
                </span>
              </motion.div>
            ))
          )}
        </AnimatePresence>

        {/* Blinking cursor */}
        <div className="flex items-center gap-1 text-signal-cyan pt-1">
          <span>{'>'}</span>
          <span className="w-2 h-4 bg-signal-cyan animate-pulse" />
        </div>
      </div>
    </div>
  )
}

// Compact inline version
export function NetworkEventBadge({ event }: { event: NetworkEvent }) {
  const getColor = () => {
    switch (event.type) {
      case 'CONNECTED': return 'bg-signal-green/20 text-signal-green border-signal-green/30'
      case 'FAILED': return 'bg-signal-red/20 text-signal-red border-signal-red/30'
      case 'PUNCH': return 'bg-signal-orange/20 text-signal-orange border-signal-orange/30'
      default: return 'bg-signal-cyan/20 text-signal-cyan border-signal-cyan/30'
    }
  }

  return (
    <span className={`inline-flex items-center gap-1.5 px-2 py-1 rounded text-xs font-mono border ${getColor()}`}>
      <span className="uppercase font-semibold">{event.type}</span>
      <span className="opacity-70">{event.message}</span>
    </span>
  )
}