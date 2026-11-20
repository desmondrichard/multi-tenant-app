import axios from 'axios';
import { store } from '../store';
import { forceLogout } from './session';

export const api = axios.create({
    baseURL: import.meta.env.VITE_API_URL || 'http://localhost:8080/api',
    headers: {
        'Content-Type': 'application/json',
    },
});

// Request Interceptor: Runs before every request
api.interceptors.request.use(
    (config) => {
        // Get token from Redux store
        const token = store.getState().auth.token;
        if (token) {
            config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
    },
    (error) => Promise.reject(error)
);

// Response Interceptor: Runs after every response
api.interceptors.response.use(
    (response) => response,
    (error) => {
        const status = error.response?.status;
        const isAuthCall = error.config?.url?.startsWith('/auth/');

        // 401 on a normal API call = token expired/invalid OR the user was
        // removed from the workspace. (Wrong password on /auth/login is also
        // a 401, so those calls are skipped.)
        if (status === 401 && !isAuthCall) {
            const code = error.response?.data?.code;
            forceLogout(code === 'ACCESS_REVOKED' ? 'removed' : 'expired');
        }
        return Promise.reject(error);
    }
);