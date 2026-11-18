import axios from 'axios';
import { store } from '../store';
import { logout } from '../store/authSlice';

export const api = axios.create({
    baseURL: import.meta.env.VITE_API_URL ||'http://localhost:8080/api',
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
        // If backend returns 401 Unauthorized (token expired/invalid)
        if (error.response?.status === 401) {
            store.dispatch(logout());
            window.location.href = '/login';
        }
        return Promise.reject(error);
    }
);