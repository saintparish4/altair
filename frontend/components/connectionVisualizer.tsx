'use client'

import { useEffect, useRef, useState, useMemo } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Wifi, Server, Globe } from 'lucide-react'

interface Peer {
  id: string
  name: string
  endpoint?: { ip: string; port: number }
  x: number
  y: number
  type: 'local' | 'remote' | 'server'
}

interface Connection {
  id: string
  from: string
  to: string
  status: 'pending' | 'connecting' | 'punching' | 'active' | 'failed'
  latency?: number
}

interface ConnectionVisualizerProps {
  peers: Peer[]
  connections: Connection[]
  showLabels?: boolean
  className?: string
}

// Smart layout positions based on peer count
function calculatePeerPositions(peerCount: number): Array<{ x: number; y: number }> {
  if (peerCount === 0) return []
  if (peerCount === 1) return [{ x: 0.5, y: 0.5 }]
  if (peerCount === 2) return [{ x: 0.25, y: 0.5 }, { x: 0.75, y: 0.5 }]
  if (peerCount === 3) {
    // Triangle layout
    return [
      { x: 0.5, y: 0.22 },   // top center
      { x: 0.2, y: 0.72 },   // bottom left
      { x: 0.8, y: 0.72 },   // bottom right
    ]
  }
  if (peerCount === 4) {
    // Square layout
    return [
      { x: 0.25, y: 0.25 },
      { x: 0.75, y: 0.25 },
      { x: 0.25, y: 0.75 },
      { x: 0.75, y: 0.75 },
    ]
  }
  
  // For 5+ peers, use circular layout
  const positions: Array<{ x: number; y: number }> = []
  for (let i = 0; i < peerCount; i++) {
    const angle = (i / peerCount) * 2 * Math.PI - Math.PI / 2
    const radius = 0.35
    positions.push({
      x: 0.5 + radius * Math.cos(angle),
      y: 0.5 + radius * Math.sin(angle),
    })
  }
  return positions
}

export function ConnectionVisualizer({
  peers,
  connections,
  showLabels = true,
  className = '',
}: ConnectionVisualizerProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const [dimensions, setDimensions] = useState({ width: 600, height: 400 })

  useEffect(() => {
    const updateDimensions = () => {
      if (containerRef.current) {
        const rect = containerRef.current.getBoundingClientRect()
        if (rect.width > 0 && rect.height > 0) {
          setDimensions({ width: rect.width, height: rect.height })
        }
      }
    }

    updateDimensions()
    const timeoutId = setTimeout(updateDimensions, 100)
    
    const resizeObserver = new ResizeObserver(updateDimensions)
    if (containerRef.current) {
      resizeObserver.observe(containerRef.current)
    }

    return () => {
      resizeObserver.disconnect()
      clearTimeout(timeoutId)
    }
  }, [])

  // Calculate positions with smart layout
  const peerPositions = useMemo(() => {
    const layoutPositions = calculatePeerPositions(peers.length)
    const padding = 80
    const safeWidth = Math.max(dimensions.width - padding * 2, 100)
    const safeHeight = Math.max(dimensions.height - padding * 2, 100)

    return peers.map((peer, index) => {
      const layout = layoutPositions[index] || { x: 0.5, y: 0.5 }
      return {
        peer,
        x: padding + layout.x * safeWidth,
        y: padding + layout.y * safeHeight,
      }
    })
  }, [peers, dimensions])

  const getStatusColor = (status: Connection['status']) => {
    switch (status) {
      case 'active': return '#4ade80'
      case 'connecting':
      case 'punching': return '#facc15'
      case 'pending': return '#22d3ee'
      case 'failed': return '#f87171'
      default: return '#6b7280'
    }
  }

  // Get peer position by ID
  const getPeerPos = (peerId: string) => {
    const found = peerPositions.find(p => p.peer.id === peerId)
    return found ? { x: found.x, y: found.y } : null
  }

  // Get line coordinates between two peers (center to center)
  const getConnectionLine = (fromId: string, toId: string) => {
    const from = getPeerPos(fromId)
    const to = getPeerPos(toId)
    if (!from || !to) return null

    return {
      x1: from.x,
      y1: from.y,
      x2: to.x,
      y2: to.y,
      midX: (from.x + to.x) / 2,
      midY: (from.y + to.y) / 2,
    }
  }

  return (
    <div
      ref={containerRef}
      className={`relative w-full h-full min-h-[400px] overflow-hidden ${className}`}
    >
      {/* Grid background */}
      <div className="absolute inset-0 grid-bg opacity-30" />
      
      {/* Radial glow effect */}
      <div className="absolute inset-0 bg-radial-glow opacity-30" />

      {/* SVG for connection lines */}
      <svg className="absolute inset-0 w-full h-full" style={{ overflow: 'visible' }}>
        <defs>
          {/* Glow filter for data packets */}
          <filter id="packet-glow" x="-50%" y="-50%" width="200%" height="200%">
            <feGaussianBlur stdDeviation="2" result="coloredBlur" />
            <feMerge>
              <feMergeNode in="coloredBlur" />
              <feMergeNode in="SourceGraphic" />
            </feMerge>
          </filter>
        </defs>

        {/* Connection lines - simple dotted/solid lines */}
        {connections.map((conn) => {
          const line = getConnectionLine(conn.from, conn.to)
          if (!line) return null

          const color = getStatusColor(conn.status)
          const isActive = conn.status === 'active'
          
          // Create path for animateMotion
          const pathId = `path-${conn.id}`
          const pathData = `M ${line.x1} ${line.y1} L ${line.x2} ${line.y2}`

          return (
            <g key={conn.id}>
              {/* Path definition for animation */}
              <path
                id={pathId}
                d={pathData}
                fill="none"
                stroke="none"
              />

              {/* Simple dotted/solid line */}
              <line
                x1={line.x1}
                y1={line.y1}
                x2={line.x2}
                y2={line.y2}
                stroke={color}
                strokeWidth={isActive ? 2.5 : 2}
                strokeDasharray={isActive ? 'none' : '6 4'}
                strokeLinecap="round"
                opacity={isActive ? 1 : 0.6}
              />

              {/* Animated data packets for active connections */}
              {isActive && (
                <>
                  {/* Packet going forward */}
                  <circle r="6" fill="#ef4444" filter="url(#packet-glow)">
                    <animateMotion
                      dur="1.5s"
                      repeatCount="indefinite"
                      path={`M ${line.x1} ${line.y1} L ${line.x2} ${line.y2}`}
                    />
                  </circle>
                  {/* Packet going backward */}
                  <circle r="6" fill="#6495ED" filter="url(#packet-glow)">
                    <animateMotion
                      dur="1.5s"
                      repeatCount="indefinite"
                      begin="0.75s"
                      path={`M ${line.x2} ${line.y2} L ${line.x1} ${line.y1}`}
                    />
                  </circle>
                </>
              )}

              {/* Latency label for active connections */}
              {showLabels && isActive && conn.latency && (
                <g>
                  <rect
                    x={line.midX - 22}
                    y={line.midY - 22}
                    width="44"
                    height="20"
                    rx="4"
                    fill="#0f1419"
                    fillOpacity="0.9"
                    stroke="#1f2937"
                    strokeWidth="1"
                  />
                  <text
                    x={line.midX}
                    y={line.midY - 8}
                    fill={color}
                    fontSize="11"
                    fontWeight="500"
                    textAnchor="middle"
                    fontFamily="monospace"
                  >
                    {conn.latency}ms
                  </text>
                </g>
              )}
            </g>
          )
        })}
      </svg>

      {/* Peer nodes */}
      <AnimatePresence>
        {peerPositions.map(({ peer, x, y }) => {
          const Icon = peer.type === 'server' ? Server : peer.type === 'local' ? Wifi : Globe
          const isConnected = connections.some(
            (c) => (c.from === peer.id || c.to === peer.id) && c.status === 'active'
          )
          const hasConnection = connections.some(
            (c) => c.from === peer.id || c.to === peer.id
          )

          return (
            <motion.div
              key={peer.id}
              initial={{ opacity: 0, scale: 0 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0 }}
              transition={{ type: 'spring', damping: 20, stiffness: 300 }}
              className="absolute pointer-events-none"
              style={{
                left: x,
                top: y,
                transform: 'translate(-50%, -50%)',
              }}
            >
              {/* Node circle */}
              <div
                className={`
                  relative w-16 h-16 rounded-full flex items-center justify-center
                  border-2 z-10
                  ${isConnected
                    ? 'border-signal-green bg-altair-surface'
                    : hasConnection
                    ? 'border-signal-cyan bg-altair-surface'
                    : 'border-signal-cyan/50 bg-altair-surface'
                  }
                `}
              >
                <Icon
                  className={`w-6 h-6 ${
                    isConnected ? 'text-signal-green' : 'text-signal-cyan'
                  }`}
                />
              </div>

              {/* Label */}
              {showLabels && (
                <div className="absolute top-full mt-3 left-1/2 -translate-x-1/2 text-center whitespace-nowrap z-20">
                  <div className={`text-sm font-semibold ${isConnected ? 'text-signal-green' : 'text-altair-text'}`}>
                    {peer.name}
                  </div>
                  {peer.endpoint && (
                    <div className="text-xs text-altair-text-dim font-mono mt-0.5">
                      {peer.endpoint.ip}:{peer.endpoint.port}
                    </div>
                  )}
                </div>
              )}
            </motion.div>
          )
        })}
      </AnimatePresence>

      {/* Empty state */}
      {peers.length === 0 && (
        <div className="absolute inset-0 flex items-center justify-center">
          <div className="text-center text-altair-text-dim">
            <Globe className="w-12 h-12 mx-auto mb-3 opacity-30" />
            <p className="text-sm">No peers connected</p>
            <p className="text-xs mt-1 opacity-70">Add peers to visualize connections</p>
          </div>
        </div>
      )}
    </div>
  )
}

// Simplified static version for SSR/initial render
export function ConnectionVisualizerStatic() {
  return (
    <div className="relative w-full h-[400px] flex items-center justify-center">
      <div className="absolute inset-0 grid-bg opacity-30" />
      <div className="absolute inset-0 bg-radial-glow opacity-30" />

      {/* Static nodes */}
      <div className="relative flex items-center justify-between w-full max-w-lg px-12">
        <div className="flex flex-col items-center gap-3">
          <div className="w-16 h-16 rounded-full border-2 border-signal-cyan/50 bg-altair-surface flex items-center justify-center">
            <Wifi className="w-6 h-6 text-signal-cyan" />
          </div>
          <span className="text-sm text-altair-text-dim">Peer A</span>
        </div>

        <div className="flex-1 mx-6 relative h-1">
          <div className="absolute inset-0 bg-linear-to-r from-signal-cyan via-signal-green to-signal-cyan rounded-full opacity-50" />
          <div className="absolute inset-0 bg-linear-to-r from-transparent via-white to-transparent rounded-full opacity-30 animate-pulse" />
        </div>

        <div className="flex flex-col items-center gap-3">
          <div className="w-16 h-16 rounded-full border-2 border-signal-cyan/50 bg-altair-surface flex items-center justify-center">
            <Globe className="w-6 h-6 text-signal-cyan" />
          </div>
          <span className="text-sm text-altair-text-dim">Peer B</span>
        </div>
      </div>
    </div>
  )
}