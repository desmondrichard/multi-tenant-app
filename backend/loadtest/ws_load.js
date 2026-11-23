// k6 WebSocket load test: how many simultaneous live connections can the server hold?
//
//   k6 run -e WS_URL=ws://localhost:8080/api -e BASE_URL=http://localhost:8080/api \
//          -e EMAIL=you@mail.com -e PASSWORD=secret loadtest/ws_load.js

import http from 'k6/http';
import ws from 'k6/ws';
import { check } from 'k6';
import { Counter } from 'k6/metrics';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080/api';
const WS_URL = __ENV.WS_URL || 'ws://localhost:8080/api';
const messages = new Counter('ws_messages_received');

export const options = {
    stages: [
        { duration: '20s', target: 50 },
        { duration: '40s', target: 200 }, // 200 open connections at the same time
        { duration: '20s', target: 0 },
    ],
    thresholds: {
        ws_connecting: ['p(95)<1000'], // connecting takes under 1 s for 95% of users
        checks: ['rate>0.99'],         // more than 99% got "101 Switching Protocols"
    },
};

export function setup() {
    const res = http.post(`${BASE_URL}/auth/login`, JSON.stringify({ email: __ENV.EMAIL, password: __ENV.PASSWORD }), {
        headers: { 'Content-Type': 'application/json' },
    });
    const token = res.json('token');
    if (!token) throw new Error('Login failed: ' + res.status);
    return { token };
}

export default function (data) {
    const res = ws.connect(`${WS_URL}/ws?token=${data.token}`, {}, (socket) => {
        socket.on('message', () => messages.add(1)); // events pushed by the Hub
        socket.on('error', (e) => console.error('ws error: ' + e.error()));
        socket.setTimeout(() => socket.close(), 30000); // keep each connection open for 30 s
    });
    check(res, { 'connected (101)': (r) => r && r.status === 101 });
}