import { store } from '../store';
import { logout } from '../store/authSlice';

// Clears the token and sends the user to the login page.
// reason: "removed" (access revoked) | "expired" (token invalid/expired)
export function forceLogout(reason = 'expired') {
    store.dispatch(logout());
    window.location.href = `/login?reason=${reason}`;
}