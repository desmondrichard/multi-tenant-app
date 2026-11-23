// k6 API load test for TaskFlow
//
// Run (PowerShell, from the backend folder):
//   k6 run -e BASE_URL=http://localhost:8080/api -e EMAIL=you@mail.com -e PASSWORD=secret loadtest/api_load.js
// Optional: -e WRITES=true  also creates/updates/deletes a task (soft delete) in the first project.
//
// DO NOT run it against production, and never against /ai/* (Gemini quota) or /invites (emails).

import http from 'k6/http';
import { check, group, sleep } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080/api';
const EMAIL = __ENV.EMAIL;
const PASSWORD = __ENV.PASSWORD;
const WRITES = __ENV.WRITES === 'true';

export const options = {
    stages: [
        { duration: '30s', target: 10 },  // warm-up
        { duration: '1m', target: 50 },   // normal load
        { duration: '1m', target: 100 },  // peak load
        { duration: '30s', target: 0 },   // ramp-down
    ],
    thresholds: {
        http_req_failed: ['rate<0.01'],                 // less than 1% errors
        http_req_duration: ['p(95)<500', 'p(99)<1000'], // 95% under 500 ms, 99% under 1 s
        checks: ['rate>0.99'],                          // more than 99% of checks pass
        'http_req_duration{name:GET /projects}': ['p(95)<400'],
        'http_req_duration{name:GET /analytics}': ['p(95)<600'],
    },
};

// Runs ONCE: log in a single time (bcrypt is CPU heavy, we are not testing login speed here)
export function setup() {
    if (!EMAIL || !PASSWORD) {
        throw new Error('Pass -e EMAIL=... -e PASSWORD=...');
    }
    const res = http.post(`${BASE_URL}/auth/login`, JSON.stringify({ email: EMAIL, password: PASSWORD }), {
        headers: { 'Content-Type': 'application/json' },
    });
    check(res, { 'login ok': (r) => r.status === 200 });
    const token = res.json('token');
    if (!token) throw new Error('Login failed: ' + res.status + ' ' + res.body);

    const headers = { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' };
    const projects = http.get(`${BASE_URL}/projects`, { headers }).json();
    const projectId = Array.isArray(projects) && projects.length > 0 ? projects[0].id : null;
    return { token, projectId };
}

function get(path, name, headers) {
    const res = http.get(`${BASE_URL}${path}`, { headers, tags: { name: `GET ${name}` } });
    check(res, { [`${name} is 200`]: (r) => r.status === 200 });
    return res;
}

// Runs for EVERY virtual user, in a loop
export default function (data) {
    const headers = { Authorization: `Bearer ${data.token}`, 'Content-Type': 'application/json' };

    group('read pages', () => {
        get('/me', '/me', headers);
        get('/projects?page=1&limit=9', '/projects', headers);
        get('/notifications?limit=10', '/notifications', headers);
        get('/activities?limit=5', '/activities', headers);
        get('/analytics', '/analytics', headers);
        get('/my-tasks?page=1&limit=10', '/my-tasks', headers);
    });

    if (WRITES && data.projectId) {
        group('write flow (create, move, delete)', () => {
            const create = http.post(
                `${BASE_URL}/projects/${data.projectId}/tasks`,
                JSON.stringify({ title: `k6 load task ${__VU}-${__ITER}`, description: 'created by k6', priority: 'low' }),
                { headers, tags: { name: 'POST /tasks' } },
            );
            check(create, { 'task created': (r) => r.status === 201 });
            if (create.status === 201) {
                const id = create.json('id');
                const move = http.patch(
                    `${BASE_URL}/projects/${data.projectId}/tasks/${id}/status`,
                    JSON.stringify({ status: 'in_progress' }),
                    { headers, tags: { name: 'PATCH /status' } },
                );
                check(move, { 'status changed': (r) => r.status === 200 });
                const del = http.del(`${BASE_URL}/projects/${data.projectId}/tasks/${id}`, null, {
                    headers,
                    tags: { name: 'DELETE /task' },
                });
                check(del, { 'task deleted': (r) => r.status === 200 });
            }
        });
    }

    sleep(1); // "think time": a real user does not click non-stop
}