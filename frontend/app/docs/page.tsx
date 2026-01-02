"use client";

import { motion } from "framer-motion";
import Link from "next/link";
import {
  ArrowLeft,
  Book,
  Code2,
  Network,
  Zap,
  Shield,
  Server,
  Terminal,
  Github,
  ExternalLink,
} from "lucide-react";
import { Navbar } from "@/components/navbar";

const sections = [
  {
    id: "overview",
    title: "Overview",
    icon: Book,
    content: (
      <div className="space-y-4">
        <p className="text-altair-text-dim leading-relaxed">
          Altair is a Go toolkit demonstrating NAT traversal techniques for
          establishing direct peer-to-peer connections. It implements the STUN
          protocol (RFC 5389), UDP hole punching, and WebSocket-based signaling
          for automatic peer discovery.
        </p>
        <p className="text-altair-text-dim leading-relaxed">
          The toolkit is built using a layered architecture approach, where
          each layer produces a working artifact before moving to the next,
          demonstrating systematic engineering practices.
        </p>
      </div>
    ),
  },
  {
    id: "stun",
    title: "STUN Protocol",
    icon: Network,
    content: (
      <div className="space-y-4">
        <p className="text-altair-text-dim leading-relaxed">
          STUN (Session Traversal Utilities for NAT) is used to discover the
          public IP address and port of a device behind a NAT. Our implementation
          is RFC 5389 compliant and supports multiple STUN servers.
        </p>
        <div className="terminal-card p-4 mt-4">
          <pre className="text-sm font-mono text-altair-text overflow-x-auto">
            <code>{`// Discover public endpoint
endpoint, err := stun.Discover("stun.l.google.com:19302")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Public endpoint: %s:%d\\n", endpoint.IP, endpoint.Port)`}</code>
          </pre>
        </div>
      </div>
    ),
  },
  {
    id: "hole-punching",
    title: "UDP Hole Punching",
    icon: Zap,
    content: (
      <div className="space-y-4">
        <p className="text-altair-text-dim leading-relaxed">
          UDP hole punching establishes direct peer-to-peer connections through
          NAT firewalls by coordinating simultaneous outbound UDP packets. Both
          peers send packets to each other's public endpoints at the same time,
          creating "holes" in their respective NATs.
        </p>
        <div className="terminal-card p-4 mt-4">
          <pre className="text-sm font-mono text-altair-text overflow-x-auto">
            <code>{`// Establish connection via hole punching
conn, err := holepunch.EstablishConnection(
    localEndpoint,
    remoteEndpoint,
    holepunch.WithRetries(5),
    holepunch.WithTimeout(10*time.Second),
)
if err != nil {
    log.Fatal(err)
}
defer conn.Close()`}</code>
          </pre>
        </div>
      </div>
    ),
  },
  {
    id: "signaling",
    title: "Signaling Server",
    icon: Server,
    content: (
      <div className="space-y-4">
        <p className="text-altair-text-dim leading-relaxed">
          The signaling server uses WebSockets to coordinate peer discovery and
          connection establishment. Peers join rooms and exchange endpoint
          information before attempting direct connections.
        </p>
        <div className="terminal-card p-4 mt-4">
          <pre className="text-sm font-mono text-altair-text overflow-x-auto">
            <code>{`// Join signaling room
client := signaling.NewClient("ws://server/ws")
err := client.JoinRoom("my-room", endpoint)
if err != nil {
    log.Fatal(err)
}

// Listen for peer events
client.OnPeerJoined(func(peer PeerInfo) {
    // Attempt connection
})`}</code>
          </pre>
        </div>
      </div>
    ),
  },
  {
    id: "nat-types",
    title: "NAT Types",
    icon: Shield,
    content: (
      <div className="space-y-4">
        <p className="text-altair-text-dim leading-relaxed mb-6">
          Different NAT types have varying levels of restrictiveness, affecting
          the success rate of hole punching:
        </p>
        <div className="grid gap-4">
          {[
            {
              type: "Full Cone",
              desc: "Most permissive. Any external host can send to the mapped port.",
              success: "95%",
            },
            {
              type: "Restricted Cone",
              desc: "Only hosts that the internal host has contacted can send back.",
              success: "85%",
            },
            {
              type: "Port Restricted",
              desc: "Same as Restricted Cone, but port must also match.",
              success: "70%",
            },
            {
              type: "Symmetric",
              desc: "Most restrictive. Different external ports for different destinations.",
              success: "30%",
            },
          ].map((nat, i) => (
            <motion.div
              key={nat.type}
              initial={{ opacity: 0, y: 20 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ delay: i * 0.1 }}
              className="terminal-card p-4"
            >
              <div className="flex items-start justify-between mb-2">
                <h4 className="font-semibold text-altair-text">{nat.type}</h4>
                <span className="text-xs px-2 py-1 rounded-full bg-signal-green/20 text-signal-green border border-signal-green/30">
                  {nat.success} success
                </span>
              </div>
              <p className="text-sm text-altair-text-dim">{nat.desc}</p>
            </motion.div>
          ))}
        </div>
      </div>
    ),
  },
];

export default function DocsPage() {
  return (
    <main className="min-h-screen">
      <Navbar />

      {/* Hero Section */}
      <section className="relative pt-32 pb-16 px-4 sm:px-6 lg:px-8">
        <div className="max-w-4xl mx-auto">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, ease: [0.22, 1, 0.36, 1] }}
          >
            <Link
              href="/"
              className="inline-flex items-center gap-2 text-altair-text-dim hover:text-altair-text transition-colors mb-8"
            >
              <ArrowLeft className="w-4 h-4" />
              <span>Back to Home</span>
            </Link>

            <h1 className="text-5xl sm:text-6xl font-display font-bold text-altair-text mb-6 tracking-tight">
              Documentation
            </h1>
            <p className="text-xl text-altair-text-dim/80 leading-relaxed font-light">
              Learn how to use Altair for NAT traversal and peer-to-peer
              networking in your Go applications.
            </p>
          </motion.div>
        </div>
      </section>

      {/* Content Section */}
      <section className="pb-32 px-4 sm:px-6 lg:px-8">
        <div className="max-w-4xl mx-auto">
          <div className="space-y-16">
            {sections.map((section, i) => (
              <motion.div
                key={section.id}
                id={section.id}
                initial={{ opacity: 0, y: 30 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true, margin: "-100px" }}
                transition={{ duration: 0.6, delay: i * 0.1, ease: [0.22, 1, 0.36, 1] }}
                className="scroll-mt-24"
              >
                <div className="flex items-center gap-4 mb-6">
                  <div className="w-12 h-12 rounded-xl bg-signal-cyan/10 border border-signal-cyan/30 flex items-center justify-center">
                    <section.icon className="w-6 h-6 text-signal-cyan" />
                  </div>
                  <h2 className="text-3xl font-display font-bold text-altair-text">
                    {section.title}
                  </h2>
                </div>
                <div className="ml-16">{section.content}</div>
              </motion.div>
            ))}
          </div>

          {/* Resources Section */}
          <motion.div
            initial={{ opacity: 0, y: 30 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            transition={{ duration: 0.6, delay: 0.5 }}
            className="mt-24 terminal-card p-8 bg-altair-surface/40 backdrop-blur-xl border-altair-border/50"
          >
            <h3 className="text-2xl font-display font-bold text-altair-text mb-6">
              Resources
            </h3>
            <div className="grid sm:grid-cols-2 gap-4">
              <a
                href="https://github.com/saintparish4/altair"
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-3 p-4 rounded-lg bg-altair-surface/60 border border-altair-border/50 hover:border-altair-muted transition-colors group"
              >
                <Github className="w-5 h-5 text-altair-text-dim group-hover:text-altair-text transition-colors" />
                <div>
                  <div className="font-medium text-altair-text">GitHub Repository</div>
                  <div className="text-sm text-altair-text-dim">View source code</div>
                </div>
                <ExternalLink className="w-4 h-4 text-altair-text-dim ml-auto" />
              </a>
              <Link
                href="/demo"
                className="flex items-center gap-3 p-4 rounded-lg bg-altair-surface/60 border border-altair-border/50 hover:border-altair-muted transition-colors group"
              >
                <Terminal className="w-5 h-5 text-altair-text-dim group-hover:text-altair-text transition-colors" />
                <div>
                  <div className="font-medium text-altair-text">Live Demo</div>
                  <div className="text-sm text-altair-text-dim">Try it out</div>
                </div>
                <ArrowLeft className="w-4 h-4 text-altair-text-dim ml-auto rotate-180" />
              </Link>
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

