"use client";

import { useEffect, useState, useRef } from "react";
import Link from "next/link";
import { motion } from "framer-motion";
import {
  ArrowRight,
  Github,
  Zap,
  Shield,
  Network,
  Code2,
  Terminal,
  Server,
} from "lucide-react";
import { Navbar } from "@/components/navbar";
import {
  ConnectionVisualizer,
  ConnectionVisualizerStatic,
} from "@/components/connectionVisualizer";
import { useDemoSignaling } from "@/lib/hook";

// Feature cards data
const features = [
  {
    icon: Network,
    title: "STUN Protocol",
    description:
      "RFC 5389 compliant implementation for discovering public IP endpoints behind NAT.",
    color: "signal-cyan",
  },
  {
    icon: Zap,
    title: "UDP Hole Punching",
    description:
      "Establish direct peer-to-peer connections through NAT firewalls.",
    color: "signal-green",
  },
  {
    icon: Shield,
    title: "NAT Traversal",
    description:
      "Support for Full Cone, Restricted Cone, and Port-Restricted Cone NATs.",
    color: "signal-orange",
  },
  {
    icon: Server,
    title: "Signaling Server",
    description: "WebSocket-based coordination for automatic peer discovery.",
    color: "signal-purple",
  },
];

// Architecture layers
const layers = [
  { name: "Layer 1", title: "STUN Client", status: "complete" },
  { name: "Layer 2", title: "Hole Punching", status: "complete" },
  { name: "Layer 3", title: "Signaling", status: "complete" },
  { name: "Layer 4", title: "Relay Fallback", status: "planned" },
];

export default function HomePage() {
  const [mounted, setMounted] = useState(false);
  const demo = useDemoSignaling();
  const initialized = useRef(false);

  useEffect(() => {
    // Only run once on mount
    if (initialized.current) return;
    initialized.current = true;

    // Set mounted state after initial render
    const mountTimeout = setTimeout(() => {
      setMounted(true);
    }, 0);

    // Start demo animation
    const timeout1 = setTimeout(() => {
      demo.simulatePeerJoin("Peer A", { ip: "203.0.113.42", port: 54321 });
    }, 1000);

    const timeout2 = setTimeout(() => {
      demo.simulatePeerJoin("Peer B", { ip: "198.51.100.17", port: 12345 });
    }, 2000);

    const timeout3 = setTimeout(() => {
      if (demo.peers.length >= 2) {
        demo.simulateConnection(demo.peers[0]?.peer_id, demo.peers[1]?.peer_id);
      }
    }, 3500);

    return () => {
      clearTimeout(mountTimeout);
      clearTimeout(timeout1);
      clearTimeout(timeout2);
      clearTimeout(timeout3);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Convert demo data to visualizer format
  const visualizerPeers = demo.peers.map((p, i) => {
    const peerType: "local" | "remote" = i === 0 ? "local" : "remote";
    return {
      id: p.peer_id,
      name: p.display_name || "Anonymous",
      endpoint: p.endpoint,
      x: i === 0 ? 0.2 : 0.8,
      y: 0.5,
      type: peerType,
    };
  });

  return (
    <main className="min-h-screen">
      <Navbar />

      {/* Hero Section */}
      <section className="relative pt-40 pb-32 px-4 sm:px-6 lg:px-8 overflow-hidden">
        {/* Refined animated background elements */}
        <div className="absolute inset-0 overflow-hidden pointer-events-none">
          <motion.div
            initial={{ opacity: 0, scale: 0.8 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ duration: 2, ease: "easeOut" }}
            className="absolute top-1/4 left-1/4 w-[500px] h-[500px] bg-signal-cyan/4 rounded-full blur-[120px]"
          />
          <motion.div
            initial={{ opacity: 0, scale: 0.8 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ duration: 2, delay: 0.3, ease: "easeOut" }}
            className="absolute bottom-1/4 right-1/4 w-[500px] h-[500px] bg-signal-green/4 rounded-full blur-[120px]"
          />
        </div>

        <div className="relative max-w-7xl mx-auto">
          <div className="text-center max-w-5xl mx-auto">
            {/* Refined Badge */}
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.6, ease: [0.22, 1, 0.36, 1] }}
              className="inline-flex items-center gap-2.5 px-5 py-2.5 rounded-full 
                bg-altair-surface/60 backdrop-blur-xl border border-altair-border/50 mb-12
                shadow-[0_0_0_0.5px_rgba(34,211,238,0.05)]"
            >
              <motion.span
                animate={{ scale: [1, 1.2, 1] }}
                transition={{ duration: 2, repeat: Infinity, ease: "easeInOut" }}
                className="w-2 h-2 rounded-full bg-signal-green shadow-[0_0_8px_rgba(74,222,128,0.6)]"
              />
              <span className="text-sm font-medium text-altair-text-dim tracking-wide">
                NAT Traversal Toolkit
              </span>
            </motion.div>

            {/* Refined Headline */}
            <motion.h1
              initial={{ opacity: 0, y: 30 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.8, delay: 0.1, ease: [0.22, 1, 0.36, 1] }}
              className="text-6xl sm:text-7xl lg:text-8xl font-display font-bold 
                tracking-[-0.02em] leading-[1.1] mb-8"
            >
              <span className="text-altair-text">Establish </span>
              <span 
                className="text-gradient bg-clip-text text-transparent"
                style={{
                  backgroundImage: 'linear-gradient(to right, #22d3ee, #4ade80, #22d3ee)',
                  backgroundSize: '200% auto',
                  animation: 'gradient 8s ease infinite'
                }}
              >
                Direct P2P
              </span>
              <br />
              <span className="text-altair-text">Connections</span>
            </motion.h1>

            {/* Refined Subheadline */}
            <motion.p
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.8, delay: 0.2, ease: [0.22, 1, 0.36, 1] }}
              className="mt-8 text-xl sm:text-2xl text-altair-text-dim/90 max-w-3xl mx-auto 
                leading-relaxed font-light tracking-wide"
            >
              A Go toolkit demonstrating NAT traversal techniques: STUN protocol
              implementation, UDP hole punching, and WebSocket signaling for
              peer-to-peer networking.
            </motion.p>

            {/* Refined CTA Buttons */}
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.8, delay: 0.3, ease: [0.22, 1, 0.36, 1] }}
              className="mt-14 flex flex-col sm:flex-row items-center justify-center gap-4"
            >
              <motion.div whileHover={{ scale: 1.02 }} whileTap={{ scale: 0.98 }}>
                <Link
                  href="/demo"
                  className="inline-flex items-center justify-center gap-2.5 px-8 py-4 rounded-xl 
                    font-medium transition-all duration-300
                    bg-signal-cyan text-altair-bg hover:bg-signal-cyan/90
                    shadow-[0_4px_20px_rgba(34,211,238,0.3)] hover:shadow-[0_6px_30px_rgba(34,211,238,0.4)]
                    focus:outline-none focus:ring-2 focus:ring-signal-cyan focus:ring-offset-2 focus:ring-offset-altair-bg"
                >
                  <Zap className="w-4 h-4" />
                  Try Live Demo
                </Link>
              </motion.div>
              <motion.div whileHover={{ scale: 1.02 }} whileTap={{ scale: 0.98 }}>
                <a
                  href="https://github.com/saintparish4/altair"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center justify-center gap-2.5 px-8 py-4 rounded-xl 
                    font-medium transition-all duration-300
                    bg-altair-surface/60 backdrop-blur-xl border border-altair-border/50 
                    text-altair-text hover:bg-altair-surface hover:border-altair-muted
                    shadow-[0_0_0_0.5px_rgba(34,211,238,0.05)]
                    focus:outline-none focus:ring-2 focus:ring-signal-cyan focus:ring-offset-2 focus:ring-offset-altair-bg"
                >
                  <Github className="w-4 h-4" />
                  View Source
                </a>
              </motion.div>
            </motion.div>
          </div>

          {/* Refined Live Demo Preview */}
          <motion.div
            initial={{ opacity: 0, y: 50 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 1, delay: 0.5, ease: [0.22, 1, 0.36, 1] }}
            className="mt-32 terminal-card overflow-hidden
              bg-altair-surface/40 backdrop-blur-2xl border-altair-border/50
              shadow-[0_8px_32px_rgba(0,0,0,0.4),0_0_0_0.5px_rgba(34,211,238,0.08)]"
          >
            <div className="terminal-card-header bg-altair-bg/30 backdrop-blur-xl border-b border-altair-border/30">
              <div className="terminal-dot bg-conn-failed" />
              <div className="terminal-dot bg-conn-pending" />
              <div className="terminal-dot bg-conn-active" />
              <span className="ml-3 text-xs text-altair-text-dim/80 font-mono tracking-wide">
                connection-visualizer
              </span>
            </div>
            <div className="h-[450px]">
              {mounted ? (
                <ConnectionVisualizer
                  peers={visualizerPeers}
                  connections={demo.connections}
                />
              ) : (
                <ConnectionVisualizerStatic />
              )}
            </div>
          </motion.div>
        </div>
      </section>

      {/* Features Section */}
      <section className="py-32 px-4 sm:px-6 lg:px-8">
        <div className="max-w-7xl mx-auto">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true, margin: "-100px" }}
            transition={{ duration: 0.8, ease: [0.22, 1, 0.36, 1] }}
            className="text-center mb-20"
          >
            <h2 className="text-4xl sm:text-5xl font-display font-bold text-altair-text 
              tracking-tight mb-6">
              Core Features
            </h2>
            <p className="text-xl text-altair-text-dim/80 max-w-2xl mx-auto 
              font-light leading-relaxed">
              Built from scratch in Go, demonstrating low-level networking
              expertise
            </p>
          </motion.div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
            {features.map((feature, i) => (
              <motion.div
                key={feature.title}
                initial={{ opacity: 0, y: 30 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true, margin: "-50px" }}
                transition={{ duration: 0.6, delay: i * 0.08, ease: [0.22, 1, 0.36, 1] }}
                whileHover={{ y: -4 }}
                className="group relative"
              >
                <div className="terminal-card p-8 h-full
                  bg-altair-surface/40 backdrop-blur-xl border-altair-border/50
                  hover:border-altair-muted/50 hover:bg-altair-surface/60
                  transition-all duration-500 ease-out
                  shadow-[0_0_0_0.5px_rgba(34,211,238,0.05)]
                  hover:shadow-[0_8px_32px_rgba(0,0,0,0.3),0_0_0_0.5px_rgba(34,211,238,0.1)]"
                >
                  <motion.div
                    whileHover={{ scale: 1.1, rotate: 5 }}
                    transition={{ type: "spring", stiffness: 300, damping: 20 }}
                    className={`w-14 h-14 rounded-2xl 
                      bg-${feature.color}/10 border border-${feature.color}/20
                      flex items-center justify-center mb-6
                      group-hover:bg-${feature.color}/15 group-hover:border-${feature.color}/30
                      transition-all duration-500
                      shadow-[0_4px_12px_rgba(0,0,0,0.2)]`}
                  >
                    <feature.icon className={`w-7 h-7 text-${feature.color} 
                      group-hover:scale-110 transition-transform duration-500`} />
                  </motion.div>
                  <h3 className="text-xl font-semibold text-altair-text mb-3 
                    tracking-tight group-hover:text-signal-cyan/80 transition-colors duration-300">
                    {feature.title}
                  </h3>
                  <p className="text-altair-text-dim/90 leading-relaxed 
                    group-hover:text-altair-text-dim transition-colors duration-300">
                    {feature.description}
                  </p>
                </div>
              </motion.div>
            ))}
          </div>
        </div>
      </section>

      {/* Architecture Section */}
      <section className="py-32 px-4 sm:px-6 lg:px-8 bg-altair-surface/20 backdrop-blur-sm">
        <div className="max-w-7xl mx-auto">
          <div className="grid lg:grid-cols-2 gap-16 items-start">
            <motion.div
              initial={{ opacity: 0, x: -30 }}
              whileInView={{ opacity: 1, x: 0 }}
              viewport={{ once: true, margin: "-100px" }}
              transition={{ duration: 0.8, ease: [0.22, 1, 0.36, 1] }}
            >
              <h2 className="text-4xl sm:text-5xl font-display font-bold text-altair-text 
                mb-8 tracking-tight">
                Layered Architecture
              </h2>
              <p className="text-xl text-altair-text-dim/80 mb-12 leading-relaxed font-light">
                Built using the Pragmatic Build Framework—each layer produces a
                working artifact before moving to the next. This incremental
                approach demonstrates systematic engineering practices.
              </p>

              <div className="space-y-5">
                {layers.map((layer, i) => (
                  <motion.div
                    key={layer.name}
                    initial={{ opacity: 0, x: -20 }}
                    whileInView={{ opacity: 1, x: 0 }}
                    viewport={{ once: true, margin: "-50px" }}
                    transition={{ duration: 0.6, delay: i * 0.1, ease: [0.22, 1, 0.36, 1] }}
                    className="flex items-center gap-5 group"
                  >
                    <motion.div
                      whileHover={{ scale: 1.05 }}
                      className={`
                        w-14 h-14 rounded-xl flex items-center justify-center 
                        font-mono text-base font-bold transition-all duration-300
                        ${
                          layer.status === "complete"
                            ? "bg-signal-green/15 text-signal-green border border-signal-green/30 shadow-[0_4px_12px_rgba(74,222,128,0.15)]"
                            : "bg-altair-border/30 text-altair-text-dim/50 border border-altair-border/50"
                        }
                      `}
                    >
                      {i + 1}
                    </motion.div>
                    <div className="flex-1">
                      <div className="flex items-center gap-3 mb-1">
                        <span className="font-semibold text-altair-text text-lg tracking-tight">
                          {layer.title}
                        </span>
                        {layer.status === "complete" && (
                          <motion.span
                            initial={{ opacity: 0, scale: 0.8 }}
                            whileInView={{ opacity: 1, scale: 1 }}
                            viewport={{ once: true }}
                            transition={{ delay: i * 0.1 + 0.2 }}
                            className="px-3 py-1 text-[10px] font-semibold uppercase tracking-widest 
                            bg-signal-green/15 text-signal-green rounded-full border border-signal-green/20"
                          >
                            Complete
                          </motion.span>
                        )}
                      </div>
                      <span className="text-xs text-altair-text-dim/70 font-mono tracking-wider">
                        {layer.name}
                      </span>
                    </div>
                  </motion.div>
                ))}
              </div>
            </motion.div>

            {/* Refined Code snippet */}
            <motion.div
              initial={{ opacity: 0, x: 30 }}
              whileInView={{ opacity: 1, x: 0 }}
              viewport={{ once: true, margin: "-100px" }}
              transition={{ duration: 0.8, ease: [0.22, 1, 0.36, 1] }}
              className="terminal-card
                bg-altair-surface/40 backdrop-blur-xl border-altair-border/50
                shadow-[0_8px_32px_rgba(0,0,0,0.3),0_0_0_0.5px_rgba(34,211,238,0.08)]"
            >
              <div className="terminal-card-header bg-altair-bg/30 backdrop-blur-xl border-b border-altair-border/30">
                <div className="terminal-dot bg-conn-failed" />
                <div className="terminal-dot bg-conn-pending" />
                <div className="terminal-dot bg-conn-active" />
                <span className="ml-3 text-xs text-altair-text-dim/80 font-mono tracking-wide">
                  main.go
                </span>
              </div>
              <pre className="p-6 text-sm font-mono overflow-x-auto leading-relaxed">
                <code className="text-altair-text/90">
                  {`// Discover public endpoint via STUN
endpoint, err := stun.Discover("stun.l.google.com:19302")

// Join signaling room for peer discovery
client := signaling.NewClient("ws://server/ws")
client.JoinRoom("my-room", endpoint)

// On peer discovered, initiate hole punching
conn, err := holepunch.EstablishConnection(
    localEndpoint,
    remoteEndpoint,
    holepunch.WithRetries(5),
    holepunch.WithTimeout(10*time.Second),
)

// Direct P2P connection established!
conn.Send([]byte("Hello, peer!"))`}
                </code>
              </pre>
            </motion.div>
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="py-32 px-4 sm:px-6 lg:px-8">
        <div className="max-w-4xl mx-auto text-center">
          <motion.div
            initial={{ opacity: 0, y: 30 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true, margin: "-100px" }}
            transition={{ duration: 0.8, ease: [0.22, 1, 0.36, 1] }}
          >
            <h2 className="text-4xl sm:text-5xl font-display font-bold text-altair-text mb-8 
              tracking-tight">
              See It In Action
            </h2>
            <p className="text-xl text-altair-text-dim/80 mb-12 max-w-2xl mx-auto 
              leading-relaxed font-light">
              Try the interactive demo to see NAT traversal in action, or dive
              into the source code to understand the implementation details.
            </p>
            <div className="flex flex-col sm:flex-row items-center justify-center gap-5">
              <motion.div whileHover={{ scale: 1.02 }} whileTap={{ scale: 0.98 }}>
                <Link
                  href="/demo"
                  className="inline-flex items-center justify-center gap-2.5 px-8 py-4 rounded-xl 
                    font-medium transition-all duration-300
                    bg-signal-cyan text-altair-bg hover:bg-signal-cyan/90
                    shadow-[0_4px_20px_rgba(34,211,238,0.3)] hover:shadow-[0_6px_30px_rgba(34,211,238,0.4)]"
                >
                  <Terminal className="w-4 h-4" />
                  Interactive Demo
                  <ArrowRight className="w-4 h-4" />
                </Link>
              </motion.div>
              <motion.div whileHover={{ scale: 1.02 }} whileTap={{ scale: 0.98 }}>
                <Link
                  href="/docs"
                  className="inline-flex items-center justify-center gap-2.5 px-8 py-4 rounded-xl 
                    font-medium transition-all duration-300
                    bg-altair-surface/60 backdrop-blur-xl border border-altair-border/50 
                    text-altair-text hover:bg-altair-surface hover:border-altair-muted
                    shadow-[0_0_0_0.5px_rgba(34,211,238,0.05)]"
                >
                  <Code2 className="w-4 h-4" />
                  Read Documentation
                </Link>
              </motion.div>
            </div>
          </motion.div>
        </div>
      </section>

      {/* Footer */}
      <footer className="py-12 px-4 sm:px-6 lg:px-8 border-t border-altair-border/50">
        <div className="max-w-7xl mx-auto flex flex-col sm:flex-row items-center justify-between gap-4">
          <div className="flex items-center gap-2.5 text-altair-text-dim/80 text-sm font-light">
            <span>Built with</span>
            <span className="text-signal-cyan font-medium">Go</span>
            <span>&</span>
            <span className="text-signal-green font-medium">Next.js</span>
          </div>
          <div className="flex items-center gap-6">
            <motion.a
              whileHover={{ scale: 1.1, y: -2 }}
              whileTap={{ scale: 0.95 }}
              href="https://github.com/saintparish4/altair"
              className="text-altair-text-dim/80 hover:text-altair-text transition-colors duration-300"
              target="_blank"
              rel="noopener noreferrer"
            >
              <Github className="w-5 h-5" />
            </motion.a>
          </div>
        </div>
      </footer>
    </main>
  );
}
