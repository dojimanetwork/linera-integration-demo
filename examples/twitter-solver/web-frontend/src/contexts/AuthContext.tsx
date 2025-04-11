import React, { createContext, useContext, useState, useEffect } from 'react';
import api from '../services/api';

interface User {
  id: string;
  username: string;
  displayName: string;
  profileImageUrl: string;
}

interface AuthContextType {
  user: User | null;
  setUser: (user: User | null) => void;
  isLoading: boolean;
  isLoggingOut: boolean;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isLoggingOut, setIsLoggingOut] = useState(false);

  useEffect(() => {
    const checkAuth = async () => {
      try {
        const response = await api.checkAuth();
        if (response.status === 'success' && response.user) {
          setUser(response.user);
        } else {
          // Clear any stale auth data if not authenticated
          sessionStorage.removeItem('code_verifier');
          setUser(null);
        }
      } catch (error) {
        console.error('Error checking authentication:', error);
        // Clear auth data on error
        sessionStorage.removeItem('code_verifier');
        setUser(null);
      } finally {
        setIsLoading(false);
      }
    };

    checkAuth();
  }, []);

  const logout = async () => {
    if (isLoggingOut) return; // Prevent multiple logout attempts

    setIsLoggingOut(true);
    try {
      await api.logout();

      // Clear all auth-related data
      sessionStorage.removeItem('code_verifier');
      localStorage.removeItem('twitter_auth_state');

      // Clear user state
      setUser(null);

      // Redirect to home page
      window.location.href = '/';
    } catch (error) {
      console.error('Error during logout:', error);
      // Even if logout fails, clear local state
      sessionStorage.removeItem('code_verifier');
      localStorage.removeItem('twitter_auth_state');
      setUser(null);
      throw error;
    } finally {
      setIsLoggingOut(false);
    }
  };

  return (
      <AuthContext.Provider value={{ user, setUser, isLoading, isLoggingOut, logout }}>
        {children}
      </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};

export default AuthContext;
