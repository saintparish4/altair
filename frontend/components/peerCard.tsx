"use client";

import { useState, useEffect } from "react";
import { motion } from "framer-motion";
import { User, Wifi, Globe, Clock, Activity, MoreVertical } from "lucide-react";
import { PeerInfo } from "@/lib/signaling-client";

interface PeerCardProps {
  peer: PeerInfo;
  isCurrentUser?: boolean;
  isConnected?: boolean;
  latency?: number;
  onConnect?: () => void;
  onDisconnect?: () => void;
  className?: string;
}

export function PeerCard({
  peer,
  isCurrentUser = false,
  isConnected = false,
  latency,
  onConnect,
  onDisconnect,
  className = "",
}: PeerCardProps) {
  const [currentTime, setCurrentTime] = useState(() => Date.now());

  useEffect(() => {
    // Update time every second for the "time ago" display
    const interval = setInterval(() => {
      setCurrentTime(Date.now());
    }, 1000);

    return () => clearInterval(interval);
  }, []);

  const timeSinceJoin = peer.joined_at
    ? Math.floor((currentTime - peer.joined_at) / 1000)
    : 0;

  const formatTime = (seconds: number) => {
    if (seconds < 60) return `${seconds}s ago`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
    return `${Math.floor(seconds / 3600)}h ago`;
  };

  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -10 }}
      className={`terminal-card ${className}`}
    >
      <div className="p-4">
        <div className="flex items-start justify-between gap-3">
          {/* Avatar & Info */}
          <div className="flex items-center gap-3">
            <div
              className={`
                w-10 h-10 rounded-full flex items-center justify-center
                ${
                  isConnected
                    ? "bg-signal-green/20 border border-signal-green/50"
                    : "bg-altair-border/50 border border-altair-border"
                }
              `}
            >
              {isCurrentUser ? (
                <User
                  className={`w-5 h-5 ${
                    isConnected ? "text-signal-green" : "text-altair-text-dim"
                  }`}
                />
              ) : (
                <Globe
                  className={`w-5 h-5 ${
                    isConnected ? "text-signal-green" : "text-altair-text-dim"
                  }`}
                />
              )}
            </div>

            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <span className="font-medium text-altair-text truncate">
                  {peer.display_name || "Anonymous"}
                </span>
                {isCurrentUser && (
                  <span className="px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wider bg-signal-cyan/20 text-signal-cyan rounded">
                    You
                  </span>
                )}
              </div>
              <div className="flex items-center gap-2 mt-0.5">
                <span className="font-mono text-xs text-altair-text-dim truncate">
                  {peer.peer_id}
                </span>
              </div>
            </div>
          </div>

          {/* Status */}
          <div
            className={`status-badge ${
              isConnected ? "status-badge-active" : "status-badge-pending"
            }`}
          >
            <span
              className={`w-1.5 h-1.5 rounded-full ${
                isConnected ? "bg-conn-active" : "bg-altair-text-dim"
              }`}
            />
            {isConnected ? "Connected" : "Available"}
          </div>
        </div>

        {/* Endpoint info */}
        {peer.endpoint && (
          <div className="mt-3 pt-3 border-t border-altair-border">
            <div className="flex items-center gap-2 text-xs text-altair-text-dim">
              <Wifi className="w-3.5 h-3.5" />
              <span className="font-mono">
                {peer.endpoint.ip}:{peer.endpoint.port}
              </span>
            </div>
          </div>
        )}

        {/* Stats row */}
        <div className="mt-3 flex items-center gap-4 text-xs">
          <div className="flex items-center gap-1.5 text-altair-text-dim">
            <Clock className="w-3.5 h-3.5" />
            <span>{formatTime(timeSinceJoin)}</span>
          </div>

          {isConnected && latency && (
            <div className="flex items-center gap-1.5 text-signal-green">
              <Activity className="w-3.5 h-3.5" />
              <span>{latency}ms</span>
            </div>
          )}
        </div>

        {/* Actions */}
        {!isCurrentUser && (
          <div className="mt-3 pt-3 border-t border-altair-border">
            {isConnected ? (
              <button
                onClick={onDisconnect}
                className="w-full py-2 px-3 text-sm font-medium text-conn-failed bg-conn-failed/10 
                  border border-conn-failed/30 rounded-md hover:bg-conn-failed/20 transition-colors"
              >
                Disconnect
              </button>
            ) : (
              <button
                onClick={onConnect}
                className="w-full py-2 px-3 text-sm font-medium text-signal-cyan bg-signal-cyan/10 
                  border border-signal-cyan/30 rounded-md hover:bg-signal-cyan/20 transition-colors"
              >
                Connect
              </button>
            )}
          </div>
        )}
      </div>
    </motion.div>
  );
}

// Compact version for lists
export function PeerCardCompact({
  peer,
  isConnected = false,
  onClick,
}: {
  peer: PeerInfo;
  isConnected?: boolean;
  onClick?: () => void;
}) {
  return (
    <motion.button
      initial={{ opacity: 0, x: -10 }}
      animate={{ opacity: 1, x: 0 }}
      exit={{ opacity: 0, x: 10 }}
      onClick={onClick}
      className="w-full flex items-center gap-3 p-3 rounded-lg bg-altair-surface/50 
        border border-altair-border hover:bg-altair-surface hover:border-altair-muted
        transition-all duration-200 text-left group"
    >
      <div
        className={`
          w-8 h-8 rounded-full flex items-center justify-center shrink-0
          ${
            isConnected
              ? "bg-signal-green/20 border border-signal-green/50"
              : "bg-altair-border/50 border border-altair-border"
          }
        `}
      >
        <Globe
          className={`w-4 h-4 ${
            isConnected ? "text-signal-green" : "text-altair-text-dim"
          }`}
        />
      </div>

      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="font-medium text-sm text-altair-text truncate">
            {peer.display_name || "Anonymous"}
          </span>
          <span
            className={`w-1.5 h-1.5 rounded-full ${
              isConnected ? "bg-conn-active" : "bg-altair-text-dim"
            }`}
          />
        </div>
        {peer.endpoint && (
          <span className="font-mono text-xs text-altair-text-dim">
            {peer.endpoint.ip}:{peer.endpoint.port}
          </span>
        )}
      </div>

      <MoreVertical className="w-4 h-4 text-altair-muted opacity-0 group-hover:opacity-100 transition-opacity" />
    </motion.button>
  );
}
