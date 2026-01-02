"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import {
  SignalingClient,
  ConnectionState,
  Message,
  PeerInfo,
  getSignalingClient,
} from "./signaling-client";

export function useSignaling(serverUrl?: string) {
  const [state, setState] = useState<ConnectionState>({
    status: "disconnected",
  });
  const [peers, setPeers] = useState<PeerInfo[]>([]);
  const [messages, setMessages] = useState<Message[]>([]);
  const clientRef = useRef<SignalingClient | null>(null);

  useEffect(() => {
    const client = getSignalingClient(serverUrl);
    clientRef.current = client;

    const unsubState = client.onStateChange(setState);
    const unsubMessage = client.onMessage((msg) => {
      setMessages((prev) => [...prev.slice(-99), msg]); // Keep last 100 messages

      // Update peer list on relevant events
      if (msg.type === "PEER_LIST" || msg.type === "ACK") {
        if (msg.payload && typeof msg.payload === "object" && "peers" in msg.payload) {
          setPeers((msg.payload as { peers: PeerInfo[] }).peers);
        }
      } else if (msg.type === "PEER_JOINED") {
        if (msg.payload && typeof msg.payload === "object") {
          setPeers((prev) => [...prev, msg.payload as PeerInfo]);
        }
      } else if (msg.type === "PEER_LEFT") {
        setPeers((prev) => prev.filter((p) => p.peer_id !== msg.peer_id));
      }
    });

    return () => {
      unsubState();
      unsubMessage();
    };
  }, [serverUrl]);

  const connect = useCallback(async () => {
    await clientRef.current?.connect();
  }, []);

  const disconnect = useCallback(() => {
    clientRef.current?.disconnect();
  }, []);

  const joinRoom = useCallback(async (roomId: string, displayName?: string) => {
    const peerList = await clientRef.current?.joinRoom(roomId, displayName);
    if (peerList) {
      setPeers(peerList);
    }
    return peerList;
  }, []);

  const leaveRoom = useCallback(async () => {
    await clientRef.current?.leaveRoom();
    setPeers([]);
  }, []);

  const sendOffer = useCallback(
    (
      targetId: string,
      endpoint: { ip: string; port: number },
      sessionId: string
    ) => {
      clientRef.current?.sendOffer(targetId, endpoint, sessionId);
    },
    []
  );

  const sendAnswer = useCallback(
    (
      targetId: string,
      endpoint: { ip: string; port: number },
      sessionId: string,
      accepted: boolean
    ) => {
      clientRef.current?.sendAnswer(targetId, endpoint, sessionId, accepted);
    },
    []
  );

  const clearMessages = useCallback(() => {
    setMessages([]);
  }, []);

  return {
    state,
    peers,
    messages,
    peerId: state.peerId,
    roomId: state.roomId,
    isConnected: state.status === "connected",
    connect,
    disconnect,
    joinRoom,
    leaveRoom,
    sendOffer,
    sendAnswer,
    clearMessages,
  };
}

// Hook for simulated demo mode (no actual server connection)
export function useDemoSignaling() {
  const [peers, setPeers] = useState<PeerInfo[]>([]);
  const [connections, setConnections] = useState<
    Array<{
      id: string;
      from: string;
      to: string;
      status: "pending" | "connecting" | "punching" | "active" | "failed";
      latency?: number;
      startedAt: number;
    }>
  >([]);
  const [events, setEvents] = useState<
    Array<{
      id: string;
      type: string;
      message: string;
      timestamp: number;
    }>
  >([]);

  const addEvent = useCallback((type: string, message: string) => {
    setEvents((prev) => [
      ...prev.slice(-49),
      {
        id: `evt-${Date.now()}`,
        type,
        message,
        timestamp: Date.now(),
      },
    ]);
  }, []);

  const simulatePeerJoin = useCallback(
    (displayName: string, endpoint?: { ip: string; port: number }) => {
      const peerId = `peer-${Math.random().toString(36).substr(2, 8)}`;
      const newPeer: PeerInfo = {
        peer_id: peerId,
        display_name: displayName,
        endpoint: endpoint || {
          ip: `${Math.floor(Math.random() * 256)}.${Math.floor(
            Math.random() * 256
          )}.${Math.floor(Math.random() * 256)}.${Math.floor(
            Math.random() * 256
          )}`,
          port: 10000 + Math.floor(Math.random() * 50000),
        },
        joined_at: Date.now(),
      };
      setPeers((prev) => [...prev, newPeer]);
      addEvent(
        "PEER_JOINED",
        `${displayName} joined (${newPeer.endpoint?.ip}:${newPeer.endpoint?.port})`
      );
      return peerId;
    },
    [addEvent]
  );

  const simulatePeerLeave = useCallback(
    (peerId: string) => {
      setPeers((prev) => {
        const peer = prev.find((p) => p.peer_id === peerId);
        if (peer) {
          addEvent("PEER_LEFT", `${peer.display_name} left`);
        }
        return prev.filter((p) => p.peer_id !== peerId);
      });
      setConnections((prev) =>
        prev.filter((c) => c.from !== peerId && c.to !== peerId)
      );
    },
    [addEvent]
  );

  const simulateConnection = useCallback(
    (fromId: string, toId: string) => {
      const connId = `conn-${Date.now()}`;
      const fromPeer = peers.find((p) => p.peer_id === fromId);
      const toPeer = peers.find((p) => p.peer_id === toId);

      if (!fromPeer || !toPeer) return;

      // Add connection in pending state
      setConnections((prev) => [
        ...prev,
        {
          id: connId,
          from: fromId,
          to: toId,
          status: "pending",
          startedAt: Date.now(),
        },
      ]);
      addEvent(
        "OFFER",
        `${fromPeer.display_name} → ${toPeer.display_name}: Sending offer`
      );

      // Simulate connection progression
      setTimeout(() => {
        setConnections((prev) =>
          prev.map((c) =>
            c.id === connId ? { ...c, status: "connecting" } : c
          )
        );
        addEvent(
          "ANSWER",
          `${toPeer.display_name} → ${fromPeer.display_name}: Accepted offer`
        );
      }, 500);

      setTimeout(() => {
        setConnections((prev) =>
          prev.map((c) => (c.id === connId ? { ...c, status: "punching" } : c))
        );
        addEvent("PUNCH", `UDP hole punching in progress...`);
      }, 1000);

      setTimeout(() => {
        const success = Math.random() > 0.1; // 90% success rate
        setConnections((prev) =>
          prev.map((c) =>
            c.id === connId
              ? {
                  ...c,
                  status: success ? "active" : "failed",
                  latency: success
                    ? 20 + Math.floor(Math.random() * 80)
                    : undefined,
                }
              : c
          )
        );
        if (success) {
          addEvent("CONNECTED", `Direct P2P connection established!`);
        } else {
          addEvent("FAILED", `Connection failed - incompatible NAT types`);
        }
      }, 2500);

      return connId;
    },
    [peers, addEvent]
  );

  const simulateFailedConnection = useCallback(
    (fromId: string, toId: string) => {
      const connId = `conn-${Date.now()}`;
      const fromPeer = peers.find((p) => p.peer_id === fromId);
      const toPeer = peers.find((p) => p.peer_id === toId);

      if (!fromPeer || !toPeer) return;

      // Add connection in pending state
      setConnections((prev) => [
        ...prev,
        {
          id: connId,
          from: fromId,
          to: toId,
          status: "pending",
          startedAt: Date.now(),
        },
      ]);
      addEvent(
        "OFFER",
        `${fromPeer.display_name} → ${toPeer.display_name}: Sending offer`
      );

      // Simulate connection progression
      setTimeout(() => {
        setConnections((prev) =>
          prev.map((c) =>
            c.id === connId ? { ...c, status: "connecting" } : c
          )
        );
        addEvent(
          "ANSWER",
          `${toPeer.display_name} → ${fromPeer.display_name}: Accepted offer`
        );
      }, 500);

      setTimeout(() => {
        setConnections((prev) =>
          prev.map((c) => (c.id === connId ? { ...c, status: "punching" } : c))
        );
        addEvent("PUNCH", `UDP hole punching in progress...`);
      }, 1000);

      // Force failure after delay
      setTimeout(() => {
        setConnections((prev) =>
          prev.map((c) =>
            c.id === connId
              ? {
                  ...c,
                  status: "failed",
                }
              : c
          )
        );
        addEvent("FAILED", `Connection failed - incompatible NAT types or firewall blocking`);
      }, 2500);

      return connId;
    },
    [peers, addEvent]
  );

  const reset = useCallback(() => {
    setPeers([]);
    setConnections([]);
    setEvents([]);
  }, []);

  return {
    peers,
    connections,
    events,
    simulatePeerJoin,
    simulatePeerLeave,
    simulateConnection,
    simulateFailedConnection,
    reset,
    addEvent,
  };
}
