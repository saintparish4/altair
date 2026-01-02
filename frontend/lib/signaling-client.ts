// Signaling client for connecting to the Altair signaling server

export type MessageType =
  | "JOIN"
  | "LEAVE"
  | "OFFER"
  | "ANSWER"
  | "CANDIDATE"
  | "DISCOVER"
  | "KEEP_ALIVE"
  | "PEER_JOINED"
  | "PEER_LEFT"
  | "PEER_LIST"
  | "ERROR"
  | "ACK";

export interface Endpoint {
  ip: string;
  port: number;
}

export interface PeerInfo {
  peer_id: string;
  display_name?: string;
  endpoint?: Endpoint;
  joined_at: number;
}

export interface Message {
  type: MessageType;
  peer_id?: string;
  target_id?: string;
  room_id?: string;
  payload?: unknown;
  timestamp?: number;
  request_id?: string;
}

export interface ConnectionState {
  status: "disconnected" | "connecting" | "connected" | "error";
  peerId?: string;
  roomId?: string;
  error?: string;
}

export type MessageHandler = (message: Message) => void;
export type StateChangeHandler = (state: ConnectionState) => void;

export class SignalingClient {
  private ws: WebSocket | null = null;
  private url: string;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;
  private keepAliveInterval: NodeJS.Timeout | null = null;

  private state: ConnectionState = { status: "disconnected" };
  private messageHandlers: Set<MessageHandler> = new Set();
  private stateHandlers: Set<StateChangeHandler> = new Set();
  private pendingRequests: Map<
    string,
    { resolve: (msg: Message) => void; reject: (err: Error) => void }
  > = new Map();

  constructor(url: string = "ws://localhost:8080/ws") {
    this.url = url;
  }

  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        resolve();
        return;
      }

      this.updateState({ status: "connecting" });

      try {
        this.ws = new WebSocket(this.url);

        this.ws.onopen = () => {
          this.reconnectAttempts = 0;
          this.startKeepAlive();
          // Don't update to connected yet - wait for ACK with peer ID
        };

        this.ws.onmessage = (event) => {
          try {
            const message: Message = JSON.parse(event.data);
            this.handleMessage(message);

            // First ACK contains our peer ID
            if (
              message.type === "ACK" &&
              message.peer_id &&
              this.state.status === "connecting"
            ) {
              this.updateState({
                status: "connected",
                peerId: message.peer_id,
              });
              resolve();
            }
          } catch (err) {
            console.error("Failed to parse message:", err);
          }
        };

        this.ws.onerror = (error) => {
          console.error("WebSocket error:", error);
          this.updateState({ status: "error", error: "Connection error" });
          reject(new Error("WebSocket connection failed"));
        };

        this.ws.onclose = () => {
          this.stopKeepAlive();
          this.updateState({ status: "disconnected" });
          this.attemptReconnect();
        };
      } catch (err) {
        this.updateState({ status: "error", error: String(err) });
        reject(err);
      }
    });
  }

  disconnect(): void {
    this.stopKeepAlive();
    this.reconnectAttempts = this.maxReconnectAttempts; // Prevent reconnect
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.updateState({ status: "disconnected" });
  }

  private attemptReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      return;
    }

    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);

    setTimeout(() => {
      if (this.state.status === "disconnected") {
        this.connect().catch(() => {});
      }
    }, delay);
  }

  private startKeepAlive(): void {
    this.stopKeepAlive();
    this.keepAliveInterval = setInterval(() => {
      this.send({ type: "KEEP_ALIVE" });
    }, 30000);
  }

  private stopKeepAlive(): void {
    if (this.keepAliveInterval) {
      clearInterval(this.keepAliveInterval);
      this.keepAliveInterval = null;
    }
  }

  private handleMessage(message: Message): void {
    // Handle pending request responses
    if (message.request_id && this.pendingRequests.has(message.request_id)) {
      const { resolve, reject } = this.pendingRequests.get(message.request_id)!;
      this.pendingRequests.delete(message.request_id);

      if (message.type === "ERROR") {
        const errorMessage = 
          (message.payload && typeof message.payload === "object" && "message" in message.payload)
            ? String((message.payload as { message: unknown }).message)
            : "Request failed";
        reject(new Error(errorMessage));
      } else {
        resolve(message);
      }
    }

    // Notify all handlers
    this.messageHandlers.forEach((handler) => handler(message));
  }

  private updateState(newState: Partial<ConnectionState>): void {
    this.state = { ...this.state, ...newState };
    this.stateHandlers.forEach((handler) => handler(this.state));
  }

  send(message: Partial<Message>): void {
    if (this.ws?.readyState !== WebSocket.OPEN) {
      console.warn("WebSocket not connected");
      return;
    }

    const fullMessage: Message = {
      type: message.type as MessageType,
      timestamp: Date.now(),
      ...message,
    };

    this.ws.send(JSON.stringify(fullMessage));
  }

  request(message: Partial<Message>, timeout = 10000): Promise<Message> {
    return new Promise((resolve, reject) => {
      const requestId = `req-${Date.now()}-${Math.random()
        .toString(36)
        .substr(2, 9)}`;

      const timeoutId = setTimeout(() => {
        this.pendingRequests.delete(requestId);
        reject(new Error("Request timeout"));
      }, timeout);

      this.pendingRequests.set(requestId, {
        resolve: (msg: Message) => {
          clearTimeout(timeoutId);
          resolve(msg);
        },
        reject: (err: Error) => {
          clearTimeout(timeoutId);
          reject(err);
        },
      });

      this.send({ ...message, request_id: requestId });
    });
  }

  // High-level API methods

  async joinRoom(
    roomId: string,
    displayName?: string,
    endpoint?: Endpoint
  ): Promise<PeerInfo[]> {
    const response = await this.request({
      type: "JOIN",
      room_id: roomId,
      payload: { display_name: displayName, endpoint },
    });

    this.updateState({ roomId });
    return (response.payload && typeof response.payload === "object" && "peers" in response.payload)
      ? (response.payload as { peers: PeerInfo[] }).peers
      : [];
  }

  async leaveRoom(): Promise<void> {
    await this.request({ type: "LEAVE" });
    this.updateState({ roomId: undefined });
  }

  async discoverPeers(roomId?: string): Promise<PeerInfo[]> {
    const response = await this.request({
      type: "DISCOVER",
      room_id: roomId,
    });
    return (response.payload && typeof response.payload === "object" && "peers" in response.payload)
      ? (response.payload as { peers: PeerInfo[] }).peers
      : [];
  }

  sendOffer(targetId: string, endpoint: Endpoint, sessionId: string): void {
    this.send({
      type: "OFFER",
      target_id: targetId,
      payload: {
        endpoint,
        session_id: sessionId,
        initiator_id: this.state.peerId,
      },
    });
  }

  sendAnswer(
    targetId: string,
    endpoint: Endpoint,
    sessionId: string,
    accepted: boolean
  ): void {
    this.send({
      type: "ANSWER",
      target_id: targetId,
      payload: {
        endpoint,
        session_id: sessionId,
        accepted,
      },
    });
  }

  // Event subscription

  onMessage(handler: MessageHandler): () => void {
    this.messageHandlers.add(handler);
    return () => this.messageHandlers.delete(handler);
  }

  onStateChange(handler: StateChangeHandler): () => void {
    this.stateHandlers.add(handler);
    handler(this.state); // Immediately call with current state
    return () => this.stateHandlers.delete(handler);
  }

  // Getters

  getState(): ConnectionState {
    return this.state;
  }

  getPeerId(): string | undefined {
    return this.state.peerId;
  }

  getRoomId(): string | undefined {
    return this.state.roomId;
  }

  isConnected(): boolean {
    return this.state.status === "connected";
  }
}

// Singleton instance for app-wide usage
let clientInstance: SignalingClient | null = null;

export function getSignalingClient(url?: string): SignalingClient {
  if (!clientInstance) {
    clientInstance = new SignalingClient(url);
  }
  return clientInstance;
}

export function resetSignalingClient(): void {
  if (clientInstance) {
    clientInstance.disconnect();
    clientInstance = null;
  }
}
