import { createSlice } from '@reduxjs/toolkit';

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
        setCredentials: (state, action) => {
            const { user, token } = action.payload;
            state.user = user;
            state.token = token;
            state.isAuthenticated = true;
            localStorage.setItem('auth', JSON.stringify({ user, token }));
        },
        updateToken: (state, action) => {
            // Used when switching workspaces
            state.token = action.payload.token;
            localStorage.setItem('auth', JSON.stringify({ user: state.user, token: action.payload.token }));
        },
        logout: (state) => {
            state.user = null;
            state.token = null;
            state.isAuthenticated = false;
            localStorage.removeItem('auth');
        },
    },
});

export const { setCredentials, updateToken, logout } = authSlice.actions;
export default authSlice.reducer;