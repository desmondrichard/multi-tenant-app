import { createSlice } from '@reduxjs/toolkit';

// Check if user is already logged in (from previous session)
const savedAuth = JSON.parse(localStorage.getItem('auth') || 'null');

const initialState = {
    user: savedAuth ? savedAuth.user : null,
    token: savedAuth ? savedAuth.token : null,
    isAuthenticated: savedAuth ? true : false,
};

const authSlice = createSlice({
    name: 'auth',
    initialState,
    reducers: {
        // Call this after successful login/register
        setCredentials: (state, action) => {
            const { user, token } = action.payload;
            state.user = user;
            state.token = token;
            state.isAuthenticated = true;

            // Save to localStorage so user stays logged in on refresh
            localStorage.setItem('auth', JSON.stringify({ user, token }));
        },

        // Call this to log out
        logout: (state) => {
            state.user = null;
            state.token = null;
            state.isAuthenticated = false;
            localStorage.removeItem('auth');
        },
    },
});

export const { setCredentials, logout } = authSlice.actions;
export default authSlice.reducer;