// Single source of truth for all API URL paths

export const Tickets = {
    list: () => '/tickets',
    create: () => '/tickets',
    get: (id: string) => `/tickets/${id}`,
    update: (id: string) => `/tickets/${id}`,
    delete: (id: string) => `/tickets/${id}`,
    moveStatus: (id: string) => `/tickets/${id}/status`,
} as const;

export const Proposals = {
    create: () => '/proposals',
    get: (id: string) => `/proposals/${id}`,
    approve: (id: string) => `/proposals/${id}/approve`,
} as const;

export const Runs = {
    startApproved: (proposalId: string) => `/runs/${proposalId}/start`,
    startAdHoc: () => '/runs/adhoc',
    finish: (sessionId: string) => `/runs/${sessionId}/finish`,
} as const;

export const Sessions = {
    list: () => '/sessions',
    listActive: () => '/sessions/list',
    logs: (id: string) => `/sessions/${id}/logs`,
    pane: (id: string) => `/sessions/${id}/pane`,
    switch: (id: string) => `/sessions/${id}/switch`,
} as const;

export const WebSocket = {
    global: () => '/ws?session=global',
} as const;