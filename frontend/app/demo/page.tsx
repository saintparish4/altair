"use client";

import { useState, useEffect, useCallback } from "react";
import { motion, AnimatePresence } from "framer-motion";
import {
  Play,
  RotateCcw,
  UserPlus,
  Zap,
  Settings2,
  Info,
  Wifi,
  Globe,
  Server,
  ChevronDown,
} from "lucide-react";
import { Navbar } from "@/components/navbar";
import { ConnectionVisualizer } from "@/components/connectionVisualizer";
import { NetworkMonitor } from "@/components/networkMonitor";
import { PeerCard, PeerCardCompact } from "@/components/peerCard";
import { useDemoSignaling } from "@/lib/hook";

// Predefined peer names for simulation
const peerNames = ["Alice", "Bob", "Charlie", "Diana", "Eve", "Frank"];

// NAT type descriptions
const natTypes = [
  {
    type: "Full Cone",
    description: "Most permissive, easiest to traverse",
    successRate: "95%",
  },
  {
    type: "Restricted Cone",
    description: "Requires prior outbound traffic",
    successRate: "85%",
  },
  {
    type: "Port Restricted",
    description: "Strict port matching required",
    successRate: "70%",
  },
  {
    type: "Symmetric",
    description: "Most restrictive, may require relay",
    successRate: "30%",
  },
];

export default function DemoPage() {
  const demo = useDemoSignaling();
  const [autoConnect, setAutoConnect] = useState(true);
  const [showInfo, setShowInfo] = useState(false);
  const [showNatInfo, setShowNatInfo] = useState(false);
  const [selectedPeers, setSelectedPeers] = useState<string[]>([]);
  const [viewMode, setViewMode] = useState<"compact" | "detailed">("compact");
  const [baseTime] = useState(() => Date.now());

  // Convert demo peers to visualizer format
  // Note: x/y values are ignored - visualizer uses smart layout internally
  const visualizerPeers = demo.peers.map((peer, index) => {
    const peerType: "local" | "remote" = index === 0 ? "local" : "remote";
    return {
      id: peer.peer_id,
      name: peer.display_name || "Anonymous",
      endpoint: peer.endpoint,
      x: 0,
      y: 0,
      type: peerType,
    };
  });

  // Add a new peer
  const addPeer = useCallback(() => {
    const usedNames = demo.peers.map((p) => p.display_name);
    const availableName =
      peerNames.find((n) => !usedNames.includes(n)) ||
      `Peer ${demo.peers.length + 1}`;
    const peerId = demo.simulatePeerJoin(availableName);

    // Auto-connect to existing peers if enabled
    if (autoConnect && demo.peers.length >= 1) {
      setTimeout(() => {
        const lastPeer = demo.peers[demo.peers.length - 1];
        if (lastPeer && peerId) {
          demo.simulateConnection(lastPeer.peer_id, peerId);
        }
      }, 500);
    }
  }, [demo, autoConnect]);

  // Connect selected peers
  const connectSelected = useCallback(() => {
    if (selectedPeers.length === 2) {
      demo.simulateConnection(selectedPeers[0], selectedPeers[1]);
      setSelectedPeers([]);
    }
  }, [demo, selectedPeers]);

  // Handle peer selection
  const togglePeerSelection = (peerId: string) => {
    setSelectedPeers((prev) => {
      if (prev.includes(peerId)) {
        return prev.filter((id) => id !== peerId);
      }
      if (prev.length >= 2) {
        return [prev[1], peerId];
      }
      return [...prev, peerId];
    });
  };

  // Run demo scenario
  const runScenario = useCallback(() => {
    demo.reset();

    // Add peers with delays
    setTimeout(
      () => demo.simulatePeerJoin("Alice", { ip: "203.0.113.42", port: 54321 }),
      500
    );
    setTimeout(
      () => demo.simulatePeerJoin("Bob", { ip: "198.51.100.17", port: 12345 }),
      1500
    );
    setTimeout(
      () => demo.simulatePeerJoin("Charlie", { ip: "192.0.2.99", port: 33333 }),
      2500
    );
    setTimeout(
      () => demo.simulatePeerJoin("Diana", { ip: "198.51.100.88", port: 45678 }),
      3500
    );
    
    // Successful connection: Alice -> Bob
    setTimeout(() => {
      const peers = demo.peers;
      const alice = peers.find((p) => p.display_name === "Alice");
      const bob = peers.find((p) => p.display_name === "Bob");
      if (alice && bob) {
        demo.simulateConnection(alice.peer_id, bob.peer_id);
      }
    }, 4000);
    
    // Failed connection: Charlie -> Diana (simulating incompatible NAT)
    setTimeout(() => {
      const peers = demo.peers;
      const charlie = peers.find((p) => p.display_name === "Charlie");
      const diana = peers.find((p) => p.display_name === "Diana");
      if (charlie && diana) {
        demo.simulateFailedConnection(charlie.peer_id, diana.peer_id);
      }
    }, 5500);
    
    // Another successful connection: Alice -> Charlie
    setTimeout(() => {
      const peers = demo.peers;
      const alice = peers.find((p) => p.display_name === "Alice");
      const charlie = peers.find((p) => p.display_name === "Charlie");
      if (alice && charlie) {
        demo.simulateConnection(alice.peer_id, charlie.peer_id);
      }
    }, 7000);
    
    // Another failed connection: Bob -> Diana
    setTimeout(() => {
      const peers = demo.peers;
      const bob = peers.find((p) => p.display_name === "Bob");
      const diana = peers.find((p) => p.display_name === "Diana");
      if (bob && diana) {
        demo.simulateFailedConnection(bob.peer_id, diana.peer_id);
      }
    }, 8500);
  }, [demo]);

  // Auto-run demo scenario on mount (only once)
  useEffect(() => {
    let cancelled = false;
    const timer = setTimeout(() => {
      if (!cancelled && demo.peers.length === 0) {
        runScenario();
      }
    }, 1000);
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <main className="min-h-screen pb-8">
      <Navbar />

      <div className="pt-24 px-4 sm:px-6 lg:px-8">
        <div className="max-w-7xl mx-auto">
          {/* Header */}
          <div className="mb-8">
            <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
              <div>
                <h1 className="text-3xl font-display font-bold text-altair-text">
                  Live Demo
                </h1>
                <p className="mt-1 text-altair-text-dim">
                  Interactive visualization of P2P connection establishment
                </p>
              </div>

              <div className="flex items-center gap-3">
                <button onClick={runScenario} className="btn-primary">
                  <Play className="w-4 h-4" />
                  Run Demo
                </button>
                <button onClick={demo.reset} className="btn-secondary">
                  <RotateCcw className="w-4 h-4" />
                  Reset
                </button>
              </div>
            </div>
          </div>

          {/* Main content grid */}
          <div className="grid lg:grid-cols-3 gap-6">
            {/* Visualizer - spans 2 columns */}
            <div className="lg:col-span-2 space-y-6">
              {/* Connection Visualizer */}
              <div className="terminal-card">
                <div className="terminal-card-header">
                  <div className="terminal-dot bg-conn-failed" />
                  <div className="terminal-dot bg-conn-pending" />
                  <div className="terminal-dot bg-conn-active" />
                  <span className="ml-2 text-xs text-altair-text-dim font-mono">
                    connection-visualizer
                  </span>
                  <div className="ml-auto flex items-center gap-2">
                    <span className="text-xs text-altair-text-dim">
                      {demo.peers.length} peers •{" "}
                      {
                        demo.connections.filter((c) => c.status === "active")
                          .length
                      }{" "}
                      active
                    </span>
                  </div>
                </div>
                <div className="h-[400px] lg:h-[500px]">
                  <ConnectionVisualizer
                    peers={visualizerPeers}
                    connections={demo.connections}
                  />
                </div>
              </div>

              {/* Network Monitor */}
              <NetworkMonitor
                events={demo.events.map((e) => {
                  const typeMap: Record<
                    string,
                    | "STUN"
                    | "OFFER"
                    | "ANSWER"
                    | "PUNCH"
                    | "CONNECTED"
                    | "FAILED"
                    | "PEER_JOINED"
                    | "PEER_LEFT"
                    | "INFO"
                  > = {
                    peer_joined: "PEER_JOINED",
                    peer_left: "PEER_LEFT",
                    connection_start: "OFFER",
                    connection_success: "CONNECTED",
                    connection_failed: "FAILED",
                    data_sent: "INFO",
                    data_received: "INFO",
                  };
                  return {
                    ...e,
                    type: typeMap[e.type] || "INFO",
                  };
                })}
                className="h-[250px]"
              />
            </div>

            {/* Sidebar */}
            <div className="space-y-6">
              {/* Controls */}
              <div className="terminal-card">
                <div className="terminal-card-header">
                  <Settings2 className="w-4 h-4 text-altair-text-dim" />
                  <span className="ml-2 text-xs text-altair-text-dim font-mono">
                    controls
                  </span>
                </div>
                <div className="p-4 space-y-4">
                  <button
                    onClick={addPeer}
                    className="w-full flex items-center justify-center gap-2 py-2.5 px-4 
                      bg-signal-cyan/10 border border-signal-cyan/30 text-signal-cyan
                      rounded-lg hover:bg-signal-cyan/20 transition-colors font-medium"
                  >
                    <UserPlus className="w-4 h-4" />
                    Add Peer
                  </button>

                  <button
                    onClick={connectSelected}
                    disabled={selectedPeers.length !== 2}
                    className="w-full flex items-center justify-center gap-2 py-2.5 px-4 
                      bg-signal-green/10 border border-signal-green/30 text-signal-green
                      rounded-lg hover:bg-signal-green/20 transition-colors font-medium
                      disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    <Zap className="w-4 h-4" />
                    {selectedPeers.length === 2
                      ? "Connect Selected"
                      : `Select ${2 - selectedPeers.length} more`}
                  </button>

                  <label className="flex items-center justify-between py-2">
                    <span className="text-sm text-altair-text">
                      Auto-connect new peers
                    </span>
                    <button
                      onClick={() => setAutoConnect(!autoConnect)}
                      className={`
                        w-10 h-6 rounded-full transition-colors relative
                        ${autoConnect ? "bg-signal-cyan" : "bg-altair-border"}
                      `}
                    >
                      <div
                        className={`
                        absolute top-1 w-4 h-4 rounded-full bg-white transition-transform
                        ${autoConnect ? "left-5" : "left-1"}
                      `}
                      />
                    </button>
                  </label>
                </div>
              </div>

              {/* Peer List */}
              <div className="terminal-card">
                <div className="terminal-card-header">
                  <Wifi className="w-4 h-4 text-altair-text-dim" />
                  <span className="ml-2 text-xs text-altair-text-dim font-mono">
                    peers ({demo.peers.length})
                  </span>
                  <div className="ml-auto flex items-center gap-1">
                    <button
                      onClick={() => setViewMode("compact")}
                      className={`px-2 py-1 text-xs rounded ${
                        viewMode === "compact"
                          ? "bg-signal-cyan/20 text-signal-cyan"
                          : "text-altair-text-dim"
                      }`}
                    >
                      Compact
                    </button>
                    <button
                      onClick={() => setViewMode("detailed")}
                      className={`px-2 py-1 text-xs rounded ${
                        viewMode === "detailed"
                          ? "bg-signal-cyan/20 text-signal-cyan"
                          : "text-altair-text-dim"
                      }`}
                    >
                      Detailed
                    </button>
                  </div>
                </div>
                <div className="p-3 space-y-2 max-h-[300px] overflow-y-auto">
                  <AnimatePresence>
                    {demo.peers.length === 0 ? (
                      <div className="text-center py-8 text-altair-text-dim text-sm">
                        No peers yet. Click &ldquo;Add Peer&rdquo; to start.
                      </div>
                    ) : (
                      demo.peers.map((peer, index) => {
                        const isSelected = selectedPeers.includes(peer.peer_id);
                        const isConnected = demo.connections.some(
                          (c) =>
                            (c.from === peer.peer_id ||
                              c.to === peer.peer_id) &&
                            c.status === "active"
                        );
                        const activeConnection = demo.connections.find(
                          (c) =>
                            (c.from === peer.peer_id ||
                              c.to === peer.peer_id) &&
                            c.status === "active"
                        );
                        const joinedAt = baseTime - index * 5000;

                        return viewMode === "compact" ? (
                          <motion.div
                            key={peer.peer_id}
                            initial={{ opacity: 0, y: 10 }}
                            animate={{ opacity: 1, y: 0 }}
                            exit={{ opacity: 0, y: -10 }}
                            className={
                              isSelected
                                ? "ring-2 ring-signal-cyan/50 rounded-lg"
                                : ""
                            }
                          >
                            <PeerCardCompact
                              peer={{
                                peer_id: peer.peer_id,
                                display_name: peer.display_name,
                                endpoint: peer.endpoint,
                                joined_at: joinedAt,
                              }}
                              isConnected={isConnected}
                              onClick={() => togglePeerSelection(peer.peer_id)}
                            />
                          </motion.div>
                        ) : (
                          <motion.div
                            key={peer.peer_id}
                            initial={{ opacity: 0, y: 10 }}
                            animate={{ opacity: 1, y: 0 }}
                            exit={{ opacity: 0, y: -10 }}
                            className={
                              isSelected
                                ? "ring-2 ring-signal-cyan/50 rounded-lg"
                                : ""
                            }
                          >
                            <PeerCard
                              peer={{
                                peer_id: peer.peer_id,
                                display_name: peer.display_name,
                                endpoint: peer.endpoint,
                                joined_at: joinedAt,
                              }}
                              isConnected={isConnected}
                              latency={activeConnection?.latency}
                              onConnect={() =>
                                togglePeerSelection(peer.peer_id)
                              }
                              onDisconnect={() =>
                                togglePeerSelection(peer.peer_id)
                              }
                            />
                          </motion.div>
                        );
                      })
                    )}
                  </AnimatePresence>
                </div>
              </div>

              {/* Connection Stats */}
              <div className="terminal-card">
                <div className="terminal-card-header">
                  <Zap className="w-4 h-4 text-altair-text-dim" />
                  <span className="ml-2 text-xs text-altair-text-dim font-mono">
                    connection-stats
                  </span>
                </div>
                <div className="p-4 space-y-3">
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-altair-text-dim">
                      Active Connections
                    </span>
                    <span className="font-mono text-signal-green">
                      {
                        demo.connections.filter((c) => c.status === "active")
                          .length
                      }
                    </span>
                  </div>
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-altair-text-dim">Pending</span>
                    <span className="font-mono text-signal-orange">
                      {
                        demo.connections.filter(
                          (c) =>
                            c.status === "pending" ||
                            c.status === "connecting" ||
                            c.status === "punching"
                        ).length
                      }
                    </span>
                  </div>
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-altair-text-dim">Failed</span>
                    <span className="font-mono text-signal-red">
                      {
                        demo.connections.filter((c) => c.status === "failed")
                          .length
                      }
                    </span>
                  </div>
                  <div className="pt-3 border-t border-altair-border">
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-altair-text-dim">Avg Latency</span>
                      <span className="font-mono text-altair-text">
                        {(() => {
                          const activeConns = demo.connections.filter(
                            (c) => c.status === "active" && c.latency
                          );
                          if (activeConns.length === 0) return "—";
                          const avg =
                            activeConns.reduce(
                              (sum, c) => sum + (c.latency || 0),
                              0
                            ) / activeConns.length;
                          return `${Math.round(avg)}ms`;
                        })()}
                      </span>
                    </div>
                  </div>
                </div>
              </div>

              {/* Signaling Server Status */}
              <div className="terminal-card">
                <div className="terminal-card-header">
                  <Server className="w-4 h-4 text-altair-text-dim" />
                  <span className="ml-2 text-xs text-altair-text-dim font-mono">
                    signaling-server
                  </span>
                </div>
                <div className="p-4 space-y-2">
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-altair-text-dim">Status</span>
                    <span className="flex items-center gap-2 text-signal-green">
                      <div className="w-2 h-2 rounded-full bg-signal-green animate-pulse" />
                      Online
                    </span>
                  </div>
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-altair-text-dim">Events Logged</span>
                    <span className="font-mono text-altair-text">
                      {demo.events.length}
                    </span>
                  </div>
                </div>
              </div>

              {/* NAT Types Info */}
              <button
                onClick={() => setShowNatInfo(!showNatInfo)}
                className="w-full terminal-card"
              >
                <div className="p-4 flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Globe className="w-4 h-4 text-signal-cyan" />
                    <span className="text-sm font-medium text-altair-text">
                      NAT Types
                    </span>
                  </div>
                  <ChevronDown
                    className={`w-4 h-4 text-altair-text-dim transition-transform ${
                      showNatInfo ? "rotate-180" : ""
                    }`}
                  />
                </div>

                <AnimatePresence>
                  {showNatInfo && (
                    <motion.div
                      initial={{ height: 0, opacity: 0 }}
                      animate={{ height: "auto", opacity: 1 }}
                      exit={{ height: 0, opacity: 0 }}
                      className="overflow-hidden"
                    >
                      <div className="px-4 pb-4 space-y-3">
                        {natTypes.map((nat) => (
                          <div
                            key={nat.type}
                            className="border border-altair-border rounded-lg p-3 space-y-1"
                          >
                            <div className="flex items-center justify-between">
                              <span className="text-sm font-medium text-altair-text">
                                {nat.type}
                              </span>
                              <span className="text-xs font-mono text-signal-green">
                                {nat.successRate}
                              </span>
                            </div>
                            <p className="text-xs text-altair-text-dim">
                              {nat.description}
                            </p>
                          </div>
                        ))}
                      </div>
                    </motion.div>
                  )}
                </AnimatePresence>
              </button>

              {/* Info Panel */}
              <button
                onClick={() => setShowInfo(!showInfo)}
                className="w-full terminal-card"
              >
                <div className="p-4 flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Info className="w-4 h-4 text-signal-cyan" />
                    <span className="text-sm font-medium text-altair-text">
                      How it works
                    </span>
                  </div>
                  <ChevronDown
                    className={`w-4 h-4 text-altair-text-dim transition-transform ${
                      showInfo ? "rotate-180" : ""
                    }`}
                  />
                </div>

                <AnimatePresence>
                  {showInfo && (
                    <motion.div
                      initial={{ height: 0, opacity: 0 }}
                      animate={{ height: "auto", opacity: 1 }}
                      exit={{ height: 0, opacity: 0 }}
                      className="overflow-hidden"
                    >
                      <div className="px-4 pb-4 text-sm text-altair-text-dim space-y-3">
                        <p>
                          This demo simulates the NAT traversal process that
                          Altair performs:
                        </p>
                        <ol className="list-decimal list-inside space-y-2">
                          <li>
                            Peers discover their public endpoints via STUN
                          </li>
                          <li>
                            Signaling server coordinates endpoint exchange
                          </li>
                          <li>
                            UDP hole punching establishes direct connection
                          </li>
                          <li>Data flows peer-to-peer, bypassing NAT</li>
                        </ol>
                      </div>
                    </motion.div>
                  )}
                </AnimatePresence>
              </button>
            </div>
          </div>
        </div>
      </div>
    </main>
  );
}
