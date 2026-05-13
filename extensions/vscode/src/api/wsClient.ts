import { RunCompletion } from './types';

export type WsMessageHandler = (completion: RunCompletion) => void;

export class WsClient {
    private ws: WebSocket | null = null;
    private handlers: WsMessageHandler[] = [];
    private url: string;
    private reconnectDelay = 1000;
    private maxReconnectDelay = 30000;
    private reconnectTimer: ReturnType<typeof setTimeout> | null = null;

    constructor(url: string) {
        this.url = url;
    }

    connect(): void {
        try {
            this.ws = new WebSocket(this.url);

            this.ws.onopen = () => {
                this.reconnectDelay = 1000;
            };

            this.ws.onmessage = (event) => {
                try {
                    const msg = JSON.parse(event.data) as { type: string; data: RunCompletion };
                    if (msg.type === 'run_completion') {
                        this.handlers.forEach(h => h(msg.data));
                    }
                } catch {
                    // ignore parse errors
                }
            };

            this.ws.onclose = () => {
                this.scheduleReconnect();
            };

            this.ws.onerror = () => {
                this.ws?.close();
            };
        } catch {
            this.scheduleReconnect();
        }
    }

    private scheduleReconnect(): void {
        if (this.reconnectTimer) return;
        this.reconnectTimer = setTimeout(() => {
            this.reconnectTimer = null;
            this.connect();
            this.reconnectDelay = Math.min(this.reconnectDelay * 2, this.maxReconnectDelay);
        }, this.reconnectDelay);
    }

    onCompletion(handler: WsMessageHandler): void {
        this.handlers.push(handler);
    }

    disconnect(): void {
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }
        this.ws?.close();
        this.ws = null;
    }
}