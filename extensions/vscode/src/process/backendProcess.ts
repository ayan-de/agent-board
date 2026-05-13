import { ChildProcess, spawn } from 'child_process';

export class BackendProcess {
    private process: ChildProcess | null = null;
    private port: number;
    private projectDir: string;

    constructor(port: number, projectDir = '') {
        this.port = port;
        this.projectDir = projectDir;
    }

    start(binaryPath: string): Promise<void> {
        return new Promise((resolve, reject) => {
            const args = ['--api', '--addr', `:${this.port}`];
            if (this.projectDir) {
                args.push('--project-dir', this.projectDir);
            }
            this.process = spawn(binaryPath, args, {
                stdio: ['ignore', 'pipe', 'pipe'],
                detached: false,
            });

            this.process.stdout?.on('data', (data: Buffer) => {
                console.log(`[agentboard] ${data.toString().trim()}`);
            });

            this.process.stderr?.on('data', (data: Buffer) => {
                console.error(`[agentboard] ${data.toString().trim()}`);
            });

            this.process.on('error', reject);
            this.process.on('exit', (code: number | null) => {
                if (code !== 0 && code !== null) {
                    console.error(`[agentboard] exited with code ${code}`);
                }
            });

            // Give the server a moment to start
            setTimeout(resolve, 500);
        });
    }

    stop(): Promise<void> {
        return new Promise((resolve) => {
            if (!this.process) {
                resolve();
                return;
            }
            this.process.on('exit', () => resolve());
            this.process.kill('SIGTERM');
            setTimeout(() => {
                if (this.process) {
                    this.process.kill('SIGKILL');
                }
                resolve();
            }, 3000);
        });
    }

    isRunning(): boolean {
        return this.process !== null && !this.process.killed;
    }
}