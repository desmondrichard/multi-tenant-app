import { createSlice } from '@reduxjs/toolkit';

// ------------------------------------------------------------------
// Only the JWT token is saved in localStorage (key: "token").
// The user's details (name, email, role, workspaces) are NOT stored in
// the browser. They live in Redux memory and are loaded from GET /me.
// ------------------------------------------------------------------

// Migration: older versions saved { user, token } under the key "auth".
// Move the token to the new key and delete the old data.
let legacyToken = null;
try {
    const legacy = JSON.parse(localStorage.getItem('auth') || 'null');
    legacyToken = legacy?.token || null;
} catch {
    legacyToken = null;
}
const savedToken = localStorage.getItem('token') || legacyToken;
if (savedToken) localStorage.setItem('token', savedToken);
localStorage.removeItem('auth');

const initialState = {
    user: null,                      // filled by GET /me (memory only)
    token: savedToken,
    isAuthenticated: !!savedToken,
};

const authSlice = createSlice({
    name: 'auth',
    initialState,
    reducers: {
        // Called after login / register / accept-invite. Only the token is kept.
        setCredentials: (state, action) => {
            const { token } = action.payload;
            state.token = token;
            state.user = null; // will be loaded from /me
            state.isAuthenticated = true;
            localStorage.setItem('token', token);
        },
        // Called with the response of GET /me
        setUser: (state, action) => {
            state.user = action.payload;
        },
        // Used when switching workspaces
        updateToken: (state, action) => {
            state.token = action.payload.token;
            state.user = null;
            localStorage.setItem('token', action.payload.token);
        },
        logout: (state) => {
            state.user = null;
            state.token = null;
            state.isAuthenticated = false;
            localStorage.removeItem('token');
            localStorage.removeItem('auth');
        },
    },
});

export const { setCredentials, setUser, updateToken, logout } = authSlice.actions;
export default authSlice.reducer;