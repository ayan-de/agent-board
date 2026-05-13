// Single source of truth for all API URL paths

export const Tickets = {
    list: () => '/api/tickets',
    create: () => '/api/tickets',
    get: (id: string) => `/api/tickets/${id}`,
    update: (id: string) => `/api/tickets/${id}`,
    delete: (id: string) => `/api/tickets/${id}`,
    moveStatus: (id: string) => `/api/tickets/${id}/status`,
} as const;

export const Proposals = {
    create: () => '/api/proposals',
    get: (id: string) => `/api/proposals/${id}`,
    approve: (id: string) => `/api/proposals/${id}/approve`,
} as const;

export const Runs = {
    startApproved: (proposalId: string) => `/api/runs/${proposalId}/start`,
    startAdHoc: () => '/api/runs/adhoc',
    finish: (sessionId: string) => `/api/runs/${sessionId}/finish`,
} as const;

export const Sessions = {
    list: () => '/api/sessions',
    listActive: () => '/api/sessions/list',
    logs: (id: string) => `/api/sessions/${id}/logs`,
    pane: (id: string) => `/api/sessions/${id}/pane`,
    switch: (id: string) => `/api/sessions/${id}/switch`,
} as const;

export const WebSocket = {
    global: () => '/api/ws?session=global',
} as const;